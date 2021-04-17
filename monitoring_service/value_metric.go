package monitoring_service

import "sync/atomic"

type ValueMetric struct {
	MetricInterface
	value int64
}

func (m *ValueMetric) Set(newValue int64) {
	atomic.StoreInt64(&m.value, newValue)
}

func (m *ValueMetric) Get() int64 {
	return atomic.LoadInt64(&m.value)
}
