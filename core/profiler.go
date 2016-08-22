package core

import (
	"runtime"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type Profiler interface {
	starter
	stopper
}

func NewProfiler(metricWriter MetricWriter, interval time.Duration) Profiler {
	return &profiler{
		interval:     interval,
		metricWriter: metricWriter,
		HookManager:  NewHookManager(),
	}
}

type profiler struct {
	interval     time.Duration
	metricWriter MetricWriter

	HookManager
}

func (p *profiler) Start() error {
	p.AddHook(p.interval, p.writeMetrics)
	return nil
}

// writeMetrics is called periodically by the HookManager base class and is
// responsible for building out a profiling metric and writing it into the
// metricWriter.
func (p *profiler) writeMetrics() error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	p.metricWriter.ProfilingMetric(&gatekeeper.ProfilingMetric{
		Timestamp: time.Now(),
		MemStats:  memStats,
	})

	return nil
}
