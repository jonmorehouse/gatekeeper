package shared

import (
	"fmt"
	"log"
	"time"
)

type MetricKind uint

const (
	SomeMetric MetricKind = iota + 1
	SomeOtherMetric
)

// declare names of each metric that are human readable
var metricNames map[MetricKind]string = map[MetricKind]string{
	SomeMetric:      "some_metric",
	SomeOtherMetric: "some_other_metric",
}

func (m MetricKind) String() string {
	name, ok := metricNames[m]
	if !ok {
		log.Fatal("MetricKind specified without name; this is a bug.")
	}
	return name
}

type Metric interface {
	Name() string
	Value() uint
	Duration() time.Duration

	HasCount() bool
	HasDuration() bool
}

type GeneralMetric struct {
	Kind  MetricKind
	Count uint

	Start time.Time
	End   time.Time
}

func (g *GeneralMetric) Name() string {
	return fmt.Sprintf("general|%s", g.Kind.String())
}

func (g *GeneralMetric) Duration() time.Duration {
	return g.End.Sub(g.Start)
}

func (g *GeneralMetric) HasDuration() bool {
	return g.Start != time.Time{} && g.End != time.Time{}
}

func (g *GeneralMetric) HasCount() bool {
	return g.Count != 0
}

type RequestMetric struct {
	Kind  MetricKind
	Count uint

	Start time.Time
	End   time.Time

	Upstream *Upstream
	Backend  *Backend

	Request  *Request
	Response *Response
}

func (r *RequestMetric) Name() string {
	return fmt.Sprintf("request|%s", r.Kind.String())
}

func (r *RequestMetric) Duration() time.Duration {
	return r.End.Sub(r.Start)
}

func (r *RequestMetric) HasDuration() bool {
	return r.Start != time.Time{} && r.End != time.Time{}
}

func (r *RequestMetric) HasCount() bool {
	return r.Count != 0
}
