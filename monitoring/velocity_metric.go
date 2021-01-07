package monitoring

import "sync/atomic"

type VelocityMetric struct {
	MetricInterface
	countItems int64
}

func (m *VelocityMetric) Set(countItems int64) {
	atomic.AddInt64(&m.countItems, countItems)
}

func (m *VelocityMetric) Get() int64 {
	return atomic.SwapInt64(&m.countItems, 0)
}
