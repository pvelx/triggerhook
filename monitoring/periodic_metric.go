package monitoring

import "sync/atomic"

type PeriodicMetric struct {
	MetricInterface
	value int64
}

func (m *PeriodicMetric) Set(newValue int64) {
	atomic.AddInt64(&m.value, newValue)
}

func (m *PeriodicMetric) Get() int64 {
	return atomic.SwapInt64(&m.value, 0)
}
