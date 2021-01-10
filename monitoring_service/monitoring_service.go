package monitoring_service

import (
	"github.com/imdario/mergo"
	"github.com/pvelx/triggerHook/contracts"
	"reflect"
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

	if _, ok := m.metrics[topic]; ok {
		return contracts.TopicExist
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
		return contracts.TopicIsNotInitialized
	}

	metric.Set(measurement)

	return nil
}

func (m *Monitoring) Listen(topic contracts.Topic, measurement interface{}) error {

	if _, ok := m.metrics[topic]; ok {
		return contracts.TopicExist
	}
	metric := &ValueMetric{}
	m.metrics[topic] = metric

	var convert func(interface{}) int64
	switch reflect.TypeOf(measurement).Kind() {
	case
		reflect.Array,
		reflect.Chan,
		reflect.Map,
		reflect.Slice:
		convert = func(i interface{}) int64 {
			return int64(reflect.ValueOf(i).Len())
		}

	case
		reflect.Int,
		reflect.Int32,
		reflect.Int64:
		convert = func(i interface{}) int64 {
			return reflect.ValueOf(i).Int()
		}

	default:
		return contracts.IncorrectMeasurementType
	}

	go func() {
		metric.Set(convert(measurement))
		time.Sleep(m.periodMeasure)
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
