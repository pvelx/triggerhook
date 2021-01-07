package monitoring_service

import (
	"github.com/pvelx/triggerHook/contracts"
	"time"
)

type MetricInterface interface {
	Set(measure int64)
	Get() int64
}

func New(periodMeasure time.Duration) contracts.MonitoringInterface {
	return &Monitoring{
		periodMeasure: periodMeasure,
		metrics:       make(map[string]MetricInterface),
		subscriptions: make(map[string][]*Subscription),
		eventChCap:    1000,
	}
}

type Subscription struct {
	eventCh chan contracts.MeasurementEvent
}

func (s *Subscription) Close() {
	close(s.eventCh)
}

type Monitoring struct {
	periodMeasure time.Duration
	metrics       map[string]MetricInterface
	subscriptions map[string][]*Subscription
	eventChCap    int
}

func (m *Monitoring) Init(topic string, calcType contracts.MetricType) error {
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

func (m *Monitoring) Pub(topic string, measure int64) error {

	if _, ok := m.subscriptions[topic]; !ok {
		return contracts.NoSubscribes
	}

	metric, ok := m.metrics[topic]
	if !ok {
		return contracts.NoTopic
	}

	metric.Set(measure)

	return nil
}

func (m *Monitoring) Sub(
	topic string,
	callback func(measurementEvent contracts.MeasurementEvent),
) (contracts.SubscriptionInterface, error) {

	if _, ok := m.metrics[topic]; !ok {
		return nil, contracts.NoTopic
	}

	subscription := &Subscription{
		eventCh: make(chan contracts.MeasurementEvent, m.eventChCap),
	}
	m.subscriptions[topic] = append(m.subscriptions[topic], subscription)

	go func() {
		for measure := range subscription.eventCh {
			callback(measure)
		}
	}()

	return subscription, nil
}

func (m *Monitoring) Run() {
	for {
		for topic, topicSubscriptions := range m.subscriptions {
			measure := m.metrics[topic].Get()
			now := time.Now()

			for _, subscription := range topicSubscriptions {
				subscription.eventCh <- contracts.MeasurementEvent{
					Measurement:   measure,
					Time:          now,
					PeriodMeasure: m.periodMeasure,
				}
			}
		}

		time.Sleep(m.periodMeasure)
	}
}
