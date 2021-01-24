package monitoring_service

import (
	"github.com/pvelx/triggerhook/contracts"
)

type MonitoringMock struct {
	contracts.MonitoringInterface

	/*
		You need to substitute *Mock methods to do substitute original functions
	*/
	InitMock    func(topic contracts.Topic, metricType contracts.MetricType) error
	PublishMock func(topic contracts.Topic, measurement int64) error
	ListenMock  func(topic contracts.Topic, callback func() int64) error
	RunMock     func()
}

func (m *MonitoringMock) Init(topic contracts.Topic, metricType contracts.MetricType) error {
	if m.InitMock == nil {
		return nil
	}
	return m.InitMock(topic, metricType)
}

func (m *MonitoringMock) Publish(topic contracts.Topic, measurement int64) error {
	if m.PublishMock == nil {
		return nil
	}
	return m.PublishMock(topic, measurement)
}

func (m *MonitoringMock) Listen(topic contracts.Topic, callback func() int64) error {
	if m.ListenMock == nil {
		return nil
	}
	return m.ListenMock(topic, callback)
}

func (m *MonitoringMock) Run() {
	if m.RunMock == nil {
		return
	}
	m.RunMock()
}
