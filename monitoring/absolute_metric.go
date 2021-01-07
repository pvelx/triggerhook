package monitoring

type AbsoluteMetric struct {
	MetricInterface
	value int64
}

func (m *AbsoluteMetric) Set(newValue int64) {
	m.value = newValue
}

func (m *AbsoluteMetric) Get() int64 {
	return m.value
}
