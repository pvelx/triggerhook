package monitoring

import (
	"errors"
	"github.com/pvelx/triggerHook/contracts"
	"time"
)

var (
	NoTopic      = errors.New("such topic does not exist")
	NoSubscribes = errors.New("subscribers of the topic do not exist")
)

type MetricInterface interface {
	Set(measure int64)
	Get() int64
}

func NewMonitoringService(periodSending time.Duration) contracts.MonitoringInterface {
	return &Monitoring{
		periodSending: periodSending,
		metrics:       make(map[string]MetricInterface),
		subscriptions: make(map[string][]*Subscription),
	}
}

type Subscription struct {
	topic  string
	chanel chan int64
}

func (s *Subscription) Close() {
	close(s.chanel)
}

type Monitoring struct {
	periodSending time.Duration
	metrics       map[string]MetricInterface
	subscriptions map[string][]*Subscription
}

func (m *Monitoring) Init(topic string, calcType contracts.MetricType) {
	var metric MetricInterface

	switch calcType {
	case contracts.Absolute:
		metric = &AbsoluteMetric{}
	case contracts.Periodic:
		metric = &PeriodicMetric{}
	}

	m.metrics[topic] = metric
}

func (m *Monitoring) Pub(topic string, measure int64) error {

	if _, ok := m.subscriptions[topic]; !ok {
		return NoSubscribes
	}

	metric, ok := m.metrics[topic]
	if !ok {
		return NoTopic
	}

	metric.Set(measure)

	return nil
}

func (m *Monitoring) Sub(topic string, callback func(measure int64)) (contracts.SubscriptionInterface, error) {
	if _, ok := m.metrics[topic]; !ok {
		return nil, NoTopic
	}

	subscription := &Subscription{
		chanel: make(chan int64, 1000),
	}
	m.subscriptions[topic] = append(m.subscriptions[topic], subscription)

	go func() {
		for measure := range subscription.chanel {
			callback(measure)
		}
	}()

	return subscription, nil
}

func (m *Monitoring) Run() {
	for {
		for topic, topicSubscriptions := range m.subscriptions {
			measure := m.metrics[topic].Get()

			for _, subscription := range topicSubscriptions {
				subscription.chanel <- measure
			}
		}

		time.Sleep(m.periodSending)
	}
}
