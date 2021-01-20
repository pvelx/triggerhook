package monitoring_service

type ValueMetric struct {
	MetricInterface
	value int64
}

func (m *ValueMetric) Set(newValue int64) {
	m.value = newValue
}

func (m *ValueMetric) Get() int64 {
	return m.value
}
