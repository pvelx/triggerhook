package monitoring_service

import (
	"github.com/imdario/mergo"
	"github.com/pvelx/triggerHook/contracts"
	"time"
)

type MetricInterface interface {
	Set(measure int64)
	Get() int64
}

type Options struct {
	PeriodMeasure time.Duration
	EventChCap    int

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
		EventChCap:    1000,
	}); err != nil {
		panic(err)
	}

	subscriptionChs := make(map[contracts.Topic][]chan contracts.MeasurementEvent)
	for topic, callback := range options.Subscriptions {
		topic := topic
		callback := callback
		eventCh := make(chan contracts.MeasurementEvent, options.EventChCap)
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
		eventChCap:      options.EventChCap,
	}
}

type Monitoring struct {
	periodMeasure   time.Duration
	metrics         map[contracts.Topic]MetricInterface
	subscriptionChs map[contracts.Topic][]chan contracts.MeasurementEvent
	eventChCap      int
}

func (m *Monitoring) Init(topic contracts.Topic, calcType contracts.MetricType) error {
	var metric MetricInterface

	switch calcType {
	case contracts.VelocityMetricType:
		metric = &VelocityMetric{}
	case contracts.ValueMetricType:
		metric = &ValueMetric{}
	}

	if _, ok := m.metrics[topic]; ok {
		return contracts.TopicExist
	}

	m.metrics[topic] = metric

	return nil
}

func (m *Monitoring) Pub(topic contracts.Topic, measure int64) error {

	if _, ok := m.subscriptionChs[topic]; !ok {
		return contracts.NoSubscribes
	}

	metric, ok := m.metrics[topic]
	if !ok {
		return contracts.NoTopic
	}

	metric.Set(measure)

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
