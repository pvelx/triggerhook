package monitoring_service

import (
	"time"

	"github.com/imdario/mergo"
	"github.com/pvelx/triggerhook/contracts"
)

type MetricInterface interface {
	Set(measure int64)
	Get() int64
}

type Options struct {
	PeriodMeasure time.Duration
	EventCap      int

	/*
		Subscribing to measurement events
	*/
	Subscriptions map[contracts.Topic]func(event contracts.MeasurementEvent)
}

func New(options *Options) contracts.MonitoringInterface {
	if options == nil {
		options = &Options{}
	}

	if err := mergo.Merge(options, Options{
		PeriodMeasure: 10 * time.Second,
		EventCap:      1000,
	}); err != nil {
		panic(err)
	}

	subscriptionChs := make(map[contracts.Topic][]chan contracts.MeasurementEvent)
	for topic, callback := range options.Subscriptions {
		topic := topic
		callback := callback
		eventCh := make(chan contracts.MeasurementEvent, options.EventCap)
		subscriptionChs[topic] = append(subscriptionChs[topic], eventCh)

		go func() {
			for measure := range eventCh {
				callback(measure)
			}
		}()
	}

	return &Monitoring{
		periodMeasure:   options.PeriodMeasure,
		metrics:         make(map[contracts.Topic]MetricInterface),
		subscriptionChs: subscriptionChs,
		EventCap:        options.EventCap,
	}
}

type Monitoring struct {
	periodMeasure   time.Duration
	metrics         map[contracts.Topic]MetricInterface
	subscriptionChs map[contracts.Topic][]chan contracts.MeasurementEvent
	EventCap        int
}

func (m *Monitoring) Init(topic contracts.Topic, calcType contracts.MetricType) error {

	if _, ok := m.metrics[topic]; ok {
		return contracts.MonitoringErrorTopicExist
	}

	var metric MetricInterface

	switch calcType {
	case contracts.IntegralMetricType:
		metric = &IntegralMetric{}
	case contracts.VelocityMetricType:
		metric = &VelocityMetric{}
	case contracts.ValueMetricType:
		metric = &ValueMetric{}
	}

	m.metrics[topic] = metric

	return nil
}

func (m *Monitoring) Publish(topic contracts.Topic, measurement int64) error {
	metric, ok := m.metrics[topic]
	if !ok {
		return contracts.MonitoringErrorTopicIsNotInitialized
	}

	metric.Set(measurement)

	return nil
}

func (m *Monitoring) Listen(topic contracts.Topic, callback func() int64) error {

	if _, ok := m.metrics[topic]; ok {
		return contracts.MonitoringErrorTopicExist
	}
	metric := &ValueMetric{}
	m.metrics[topic] = metric

	go func() {
		for {
			metric.Set(callback())
			time.Sleep(m.periodMeasure)
		}
	}()

	return nil
}

func (m *Monitoring) Run() {
	for {
		for topic, topicSubscriptions := range m.subscriptionChs {
			measure := m.metrics[topic].Get()
			now := time.Now()

			for _, subscriptionCh := range topicSubscriptions {
				subscriptionCh <- contracts.MeasurementEvent{
					Measurement:   measure,
					Time:          now,
					PeriodMeasure: m.periodMeasure,
				}
			}
		}

		time.Sleep(m.periodMeasure)
	}
}
