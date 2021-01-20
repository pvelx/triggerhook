package monitoring_service

import "sync/atomic"

type IntegralMetric struct {
	MetricInterface
	value int64
}

func (m *IntegralMetric) Set(newValue int64) {
	atomic.AddInt64(&m.value, newValue)
}

func (m *IntegralMetric) Get() int64 {
	return atomic.LoadInt64(&m.value)
}
