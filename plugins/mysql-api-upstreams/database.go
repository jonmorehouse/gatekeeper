package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type Database interface {
	Connect(string) error
	Disconnect() error

	AddUpstream(*shared.Upstream) (shared.UpstreamID, error)
	RemoveUpstream(shared.UpstreamID) error

	AddBackend(shared.UpstreamID, *shared.Backend) (shared.BackendID, error)
	RemoveBackend(shared.BackendID) error

	// methods used by the manager to keep local state in sync with the
	// datastore behind the scenes.
	FetchUpstreams() ([]*shared.Upstream, error)
	FetchUpstreamBackends(shared.UpstreamID) ([]*shared.Backend, error)
}

type DatabaseClient interface {
	AddUpstream(*shared.Upstream) (shared.UpstreamID, error)
	RemoveUpstream(shared.UpstreamID) error

	AddBackend(shared.UpstreamID, *shared.Backend) (shared.BackendID, error)
	RemoveBackend(shared.BackendID) error

	FetchUpstreams() ([]*shared.Upstream, error)
	FetchUpstreamBackends(shared.UpstreamID) ([]*shared.Backend, error)
}

func NewDatabase() Database {
	return &database{}
}

type database struct {
	db *sql.DB
}

func (d *database) Connect(dsn string) error {
	if d.db != nil {
		return fmt.Errorf("Already connected")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	d.db = db

	if err := d.db.Ping(); err != nil {
		return err
	}
	return nil
}

func (d *database) Disconnect() error {
	if d.db == nil {
		return fmt.Errorf("Not connected")
	}
	return d.db.Close()
}

func (d *database) AddUpstream(upstream *shared.Upstream) (shared.UpstreamID, error) {
	transaction, err := d.db.Begin()
	if err != nil {
		return shared.NilUpstreamID, err
	}

	rollback := func() {
		if err := transaction.Rollback(); err != nil {
			log.Println("unable to rollback database state after failed transaction: ", err)
		}
	}

	result, err := transaction.Exec("INSERT INTO upstream (`name`, `timeout`) VALUES (?, ?)", upstream.Name, upstream.Timeout.String())
	if err != nil {
		rollback()
		return shared.NilUpstreamID, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		rollback()
		return shared.NilUpstreamID, err
	}

	upstreamID := shared.UpstreamID(strconv.FormatInt(id, 10))

	// commit all hostnames
	for _, hostname := range upstream.Hostnames {
		_, err := transaction.Exec("INSERT INTO `upstream_mapping` (`upstream_id`, `mapping_type`, `mapping`) VALUES (?, 'hostname', ?)", upstreamID.String(), hostname)
		if err != nil {
			rollback()
			return shared.NilUpstreamID, err
		}
	}

	// commit all prefixes
	for _, prefix := range upstream.Prefixes {
		_, err := transaction.Exec("INSERT INTO `upstream_mapping` (`upstream_id`, `mapping_type`, `mapping`) VALUES (?, 'prefix', ?)", upstreamID.String(), prefix)
		if err != nil {
			rollback()
			return shared.NilUpstreamID, err
		}
	}

	// commit all protocols
	for _, protocol := range upstream.Protocols {
		_, err := transaction.Exec("INSERT INTO `upstream_protocol` (`upstream_id`, `protocol`) VALUES (?, ?)", upstreamID.String(), protocol.String())
		if err != nil {
			rollback()
			return shared.NilUpstreamID, err
		}
	}

	if err := transaction.Commit(); err != nil {
		return shared.NilUpstreamID, err
	}
	return upstreamID, nil
}

func (d *database) RemoveUpstream(upstreamID shared.UpstreamID) error {
	transaction, err := d.db.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		if err := transaction.Rollback(); err != nil {
			log.Println("Unable to rollback database after failed transaction: ", err)
		}
	}

	_, err = transaction.Exec("DELETE FROM `upstream` WHERE `id`=?", upstreamID.String())
	if err != nil {
		rollback()
		return err
	}

	_, err = transaction.Exec("DELETE FROM `upstream_mapping` WHERE `upstream_id`=?", upstreamID.String())
	if err != nil {
		rollback()
		return err
	}

	_, err = transaction.Exec("DELETE FROM `upstream_protocol` WHERE `upstream_id`=?", upstreamID.String())
	if err != nil {
		rollback()
		return err
	}

	_, err = transaction.Exec("DELETE FROM `backend` WHERE `upstream_id`=?", upstreamID.String())
	if err != nil {
		rollback()
		return err
	}

	if err := transaction.Commit(); err != nil {
		return err
	}

	return nil
}

func (d *database) AddBackend(upstreamID shared.UpstreamID, backend *shared.Backend) (shared.BackendID, error) {
	result, err := d.db.Exec("INSERT INTO `backend` (`upstream_id`, `address`, `healthcheck`) VALUES (?, ?, ?)", upstreamID.String(), backend.Address, backend.Healthcheck)
	if err != nil {
		return shared.NilBackendID, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return shared.NilBackendID, err
	}

	return shared.BackendID(strconv.FormatInt(id, 10)), nil
}

func (d *database) RemoveBackend(backendID shared.BackendID) error {
	if _, err := d.db.Exec("DELETE FROM `backend` WHERE `id`=?", backendID.String()); err != nil {
		return err
	}
	return nil
}

func (d *database) FetchUpstreams() ([]*shared.Upstream, error) {
	// map the raw ID to the upstream to make updating a little easier
	upstreams := make(map[int64]*shared.Upstream)

	rows, err := d.db.Query("SELECT `id`, `name`, `timeout` FROM `upstream`")
	if err != nil {
		return []*shared.Upstream(nil), err
	}
	for rows.Next() {
		var id int64
		var name string
		var timeout string

		if err := rows.Scan(&id, &name, &timeout); err != nil {
			return []*shared.Upstream(nil), err
		}

		parsedTimeout, err := time.ParseDuration(timeout)
		if err != nil {
			return []*shared.Upstream(nil), err
		}

		upstreams[id] = &shared.Upstream{
			Name:      name,
			ID:        shared.UpstreamID(strconv.FormatInt(id, 10)),
			Timeout:   parsedTimeout,
			Protocols: make([]shared.Protocol, 0, 0),
			Hostnames: make([]string, 0, 0),
			Prefixes:  make([]string, 0, 0),
		}
	}
	if err := rows.Err(); err != nil {
		return []*shared.Upstream(nil), err
	}

	rows, err = d.db.Query("SELECT `upstream_id`, `mapping_type`, `mapping` FROM `upstream_mapping`")
	if err != nil {
		return []*shared.Upstream(nil), err
	}
	for rows.Next() {
		var upstreamID int64
		var mappingType, mappingValue string

		if err := rows.Scan(&upstreamID, &mappingType, &mappingValue); err != nil {
			return []*shared.Upstream(nil), err
		}

		if _, ok := upstreams[upstreamID]; !ok {
			log.Println("orphaned upstream_mapping...")
			continue
		}

		if mappingType != "hostname" && mappingType != "prefix" && mappingType != "protocol" {
			return []*shared.Upstream(nil), fmt.Errorf("Invalid upstream mapping type")
		}

		if mappingType == "hostname" {
			upstreams[upstreamID].Hostnames = append(upstreams[upstreamID].Hostnames, mappingValue)
		}
		if mappingType == "prefix" {
			upstreams[upstreamID].Hostnames = append(upstreams[upstreamID].Prefixes, mappingValue)
		}
		if mappingType == "protocol" {
			protocol, err := shared.NewProtocol(mappingValue)
			if err != nil {
				return []*shared.Upstream(nil), err
			}
			upstreams[upstreamID].Protocols = append(upstreams[upstreamID].Protocols, protocol)
		}
	}
	if err := rows.Err(); err != nil {
		return []*shared.Upstream(nil), err
	}

	results := make([]*shared.Upstream, 0, len(upstreams))
	for _, upstream := range upstreams {
		results = append(results, upstream)
	}
	return results, nil
}

func (d *database) FetchUpstreamBackends(upstreamID shared.UpstreamID) ([]*shared.Backend, error) {
	backends := make([]*shared.Backend, 0, 0)
	rows, err := d.db.Query("SELECT `id`, `address`, `healthcheck` FROM `backend` WHERE `upstream_id`=?", upstreamID.String())
	if err != nil {
		return []*shared.Backend(nil), err
	}
	for rows.Next() {
		var backendID int64
		var address, healthcheck string

		if err := rows.Scan(&backendID, &address, &healthcheck); err != nil {
			return []*shared.Backend(nil), err
		}
		backends = append(backends, &shared.Backend{
			ID:          shared.BackendID(strconv.FormatInt(backendID, 10)),
			Address:     address,
			Healthcheck: healthcheck,
		})
	}
	if err := rows.Err(); err != nil {
		return []*shared.Backend(nil), err
	}

	return backends, nil
}
