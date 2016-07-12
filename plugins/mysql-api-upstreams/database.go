package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type Database interface {
	Connect(string) error
	Disconnect() error

	AddUpstream(*gatekeeper.Upstream) (gatekeeper.UpstreamID, error)
	RemoveUpstream(gatekeeper.UpstreamID) error

	AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) (gatekeeper.BackendID, error)
	RemoveBackend(gatekeeper.BackendID) error

	// methods used by the manager to keep local state in sync with the
	// datastore behind the scenes.
	FetchUpstreams() ([]*gatekeeper.Upstream, error)
	FetchUpstreamBackends(gatekeeper.UpstreamID) ([]*gatekeeper.Backend, error)
}

type DatabaseClient interface {
	AddUpstream(*gatekeeper.Upstream) (gatekeeper.UpstreamID, error)
	RemoveUpstream(gatekeeper.UpstreamID) error

	AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) (gatekeeper.BackendID, error)
	RemoveBackend(gatekeeper.BackendID) error

	FetchUpstreams() ([]*gatekeeper.Upstream, error)
	FetchUpstreamBackends(gatekeeper.UpstreamID) ([]*gatekeeper.Backend, error)
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

func (d *database) AddUpstream(upstream *gatekeeper.Upstream) (gatekeeper.UpstreamID, error) {
	transaction, err := d.db.Begin()
	if err != nil {
		return gatekeeper.NilUpstreamID, err
	}

	rollback := func() {
		if err := transaction.Rollback(); err != nil {
			log.Println("unable to rollback database state after failed transaction: ", err)
		}
	}

	result, err := transaction.Exec("INSERT INTO upstream (`name`, `timeout`) VALUES (?, ?)", upstream.Name, upstream.Timeout.String())
	if err != nil {
		rollback()
		return gatekeeper.NilUpstreamID, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		rollback()
		return gatekeeper.NilUpstreamID, err
	}

	upstreamID := gatekeeper.UpstreamID(strconv.FormatInt(id, 10))

	// commit all hostnames
	for _, hostname := range upstream.Hostnames {
		_, err := transaction.Exec("INSERT INTO `upstream_mapping` (`upstream_id`, `mapping_type`, `mapping`) VALUES (?, 'hostname', ?)", upstreamID.String(), hostname)
		if err != nil {
			rollback()
			return gatekeeper.NilUpstreamID, err
		}
	}

	// commit all prefixes
	for _, prefix := range upstream.Prefixes {
		_, err := transaction.Exec("INSERT INTO `upstream_mapping` (`upstream_id`, `mapping_type`, `mapping`) VALUES (?, 'prefix', ?)", upstreamID.String(), prefix)
		if err != nil {
			rollback()
			return gatekeeper.NilUpstreamID, err
		}
	}

	// commit all protocols
	for _, protocol := range upstream.Protocols {
		_, err := transaction.Exec("INSERT INTO `upstream_protocol` (`upstream_id`, `protocol`) VALUES (?, ?)", upstreamID.String(), protocol.String())
		if err != nil {
			rollback()
			return gatekeeper.NilUpstreamID, err
		}
	}

	if err := transaction.Commit(); err != nil {
		return gatekeeper.NilUpstreamID, err
	}
	return upstreamID, nil
}

func (d *database) RemoveUpstream(upstreamID gatekeeper.UpstreamID) error {
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

func (d *database) AddBackend(upstreamID gatekeeper.UpstreamID, backend *gatekeeper.Backend) (gatekeeper.BackendID, error) {
	result, err := d.db.Exec("INSERT INTO `backend` (`upstream_id`, `address`, `healthcheck`) VALUES (?, ?, ?)", upstreamID.String(), backend.Address, backend.Healthcheck)
	if err != nil {
		return gatekeeper.NilBackendID, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return gatekeeper.NilBackendID, err
	}

	return gatekeeper.BackendID(strconv.FormatInt(id, 10)), nil
}

func (d *database) RemoveBackend(backendID gatekeeper.BackendID) error {
	if _, err := d.db.Exec("DELETE FROM `backend` WHERE `id`=?", backendID.String()); err != nil {
		return err
	}
	return nil
}

func (d *database) FetchUpstreams() ([]*gatekeeper.Upstream, error) {
	// map the raw ID to the upstream to make updating a little easier
	upstreams := make(map[int64]*gatekeeper.Upstream)

	rows, err := d.db.Query("SELECT `id`, `name`, `timeout` FROM `upstream`")
	if err != nil {
		return []*gatekeeper.Upstream(nil), err
	}
	for rows.Next() {
		var id int64
		var name string
		var timeout string

		if err := rows.Scan(&id, &name, &timeout); err != nil {
			return []*gatekeeper.Upstream(nil), err
		}

		parsedTimeout, err := time.ParseDuration(timeout)
		if err != nil {
			return []*gatekeeper.Upstream(nil), err
		}

		upstreams[id] = &gatekeeper.Upstream{
			Name:      name,
			ID:        gatekeeper.UpstreamID(strconv.FormatInt(id, 10)),
			Timeout:   parsedTimeout,
			Protocols: make([]gatekeeper.Protocol, 0, 0),
			Hostnames: make([]string, 0, 0),
			Prefixes:  make([]string, 0, 0),
		}
	}
	if err := rows.Err(); err != nil {
		return []*gatekeeper.Upstream(nil), err
	}

	rows, err = d.db.Query("SELECT `upstream_id`, `mapping_type`, `mapping` FROM `upstream_mapping`")
	if err != nil {
		return []*gatekeeper.Upstream(nil), err
	}
	for rows.Next() {
		var upstreamID int64
		var mappingType, mappingValue string

		if err := rows.Scan(&upstreamID, &mappingType, &mappingValue); err != nil {
			return []*gatekeeper.Upstream(nil), err
		}

		if _, ok := upstreams[upstreamID]; !ok {
			log.Println("orphaned upstream_mapping...")
			continue
		}

		if mappingType != "hostname" && mappingType != "prefix" && mappingType != "protocol" {
			return []*gatekeeper.Upstream(nil), fmt.Errorf("Invalid upstream mapping type")
		}

		if mappingType == "hostname" {
			upstreams[upstreamID].Hostnames = append(upstreams[upstreamID].Hostnames, mappingValue)
		}
		if mappingType == "prefix" {
			upstreams[upstreamID].Hostnames = append(upstreams[upstreamID].Prefixes, mappingValue)
		}
		if mappingType == "protocol" {
			protocol, err := gatekeeper.NewProtocol(mappingValue)
			if err != nil {
				return []*gatekeeper.Upstream(nil), err
			}
			upstreams[upstreamID].Protocols = append(upstreams[upstreamID].Protocols, protocol)
		}
	}
	if err := rows.Err(); err != nil {
		return []*gatekeeper.Upstream(nil), err
	}

	results := make([]*gatekeeper.Upstream, 0, len(upstreams))
	for _, upstream := range upstreams {
		results = append(results, upstream)
	}
	return results, nil
}

func (d *database) FetchUpstreamBackends(upstreamID gatekeeper.UpstreamID) ([]*gatekeeper.Backend, error) {
	backends := make([]*gatekeeper.Backend, 0, 0)
	rows, err := d.db.Query("SELECT `id`, `address`, `healthcheck` FROM `backend` WHERE `upstream_id`=?", upstreamID.String())
	if err != nil {
		return []*gatekeeper.Backend(nil), err
	}
	for rows.Next() {
		var backendID int64
		var address, healthcheck string

		if err := rows.Scan(&backendID, &address, &healthcheck); err != nil {
			return []*gatekeeper.Backend(nil), err
		}
		backends = append(backends, &gatekeeper.Backend{
			ID:          gatekeeper.BackendID(strconv.FormatInt(backendID, 10)),
			Address:     address,
			Healthcheck: healthcheck,
		})
	}
	if err := rows.Err(); err != nil {
		return []*gatekeeper.Backend(nil), err
	}

	return backends, nil
}
