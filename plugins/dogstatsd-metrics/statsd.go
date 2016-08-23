package main

import (
	"fmt"
	"log"
	"strconv"
)

type Statsd interface {
	Close() error
	Count(string, int64, []string, float64) error
	Gauge(string, float64, []string, float64) error
	Histogram(string, float64, []string, float64) error
	TimeInMilliseconds(string, float64, []string, float64) error
}

func NewDebugStatsd(namespace string, tags []string) Statsd {
	return debugStatsd{
		namespace: namespace,
		tags:      tags,
	}
}

type debugStatsd struct {
	namespace string
	tags      []string
}

func (d debugStatsd) Close() error { return nil }

func (d debugStatsd) Count(key string, val int64, tags []string, _ float64) error {
	d.metric("count", key, strconv.FormatInt(val, 64), tags)
	return nil
}

func (d debugStatsd) Gauge(key string, val float64, tags []string, _ float64) error {
	d.metric("gauge", key, strconv.FormatFloat(val, 'E', -1, 64), tags)
	return nil
}

func (d debugStatsd) Histogram(key string, val float64, tags []string, _ float64) error {
	d.metric("histogram", key, strconv.FormatFloat(val, 'E', -1, 64), tags)
	return nil
}

func (d debugStatsd) TimeInMilliseconds(key string, val float64, tags []string, _ float64) error {
	d.metric("timing", key, strconv.FormatFloat(val, 'E', -1, 64)+"ms", tags)
	return nil
}

func (d debugStatsd) metric(kind, key, val string, tags []string) {
	msg := fmt.Sprintf("metric.%s.%s %s", kind, key, val)
	for _, tag := range tags {
		msg += fmt.Sprintf(" tag=%s", tag)
	}
	for _, tag := range d.tags {
		msg += fmt.Sprintf(" global.tag=%s", tag)
	}

	log.Println(msg)
}
