package monitoring_service

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMainFlow(t *testing.T) {
	tests := []struct {
		name                     string
		inputMeasurement         []int64
		expectedMeasurementEvent []contracts.MeasurementEvent
		periodInputPub           time.Duration
		periodMeasure            time.Duration
		metricType               contracts.MetricType
	}{
		{
			name:             "value metric",
			inputMeasurement: []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			expectedMeasurementEvent: []contracts.MeasurementEvent{
				{Measurement: 0},
				{Measurement: 1},
				{Measurement: 3},
				{Measurement: 5},
				{Measurement: 7},
				{Measurement: 9},
			},
			periodInputPub: 50 * time.Millisecond,
			periodMeasure:  100 * time.Millisecond,
			metricType:     contracts.ValueMetricType,
		},
		{
			name:             "velocity metric",
			inputMeasurement: []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			expectedMeasurementEvent: []contracts.MeasurementEvent{
				{Measurement: 0},
				{Measurement: 2},
				{Measurement: 10},
				{Measurement: 18},
				{Measurement: 26},
				{Measurement: 34},
			},
			periodInputPub: 50 * time.Millisecond,
			periodMeasure:  100 * time.Millisecond,
			metricType:     contracts.VelocityMetricType,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var actualMeasurementCh = make(chan contracts.MeasurementEvent)
			var topicName contracts.Topic = "topic"

			monitoringService := New(&Options{
				PeriodMeasure: test.periodMeasure,
				EventChCap:    1000000,
				Subscriptions: map[contracts.Topic]func(event contracts.MeasurementEvent){
					topicName: func(measurementEvent contracts.MeasurementEvent) {
						actualMeasurementCh <- measurementEvent

						assert.Equal(t, measurementEvent.PeriodMeasure, test.periodMeasure,
							"Unexpected period of measure")
					},
				},
			})

			monitoringService.Init(topicName, test.metricType)
			go monitoringService.Run()
			time.Sleep(10 * time.Millisecond)

			blockCh := make(chan bool)
			for worker := 0; worker < 2; worker++ {
				go func() {
					<-blockCh
					for _, measure := range test.inputMeasurement {
						if err := monitoringService.Pub(topicName, measure); err != nil {
							t.Fatal(err)
						}
						time.Sleep(test.periodInputPub)
					}
				}()
			}
			close(blockCh)

			for i := 0; i < len(test.expectedMeasurementEvent); i++ {
				assert.Equal(t,
					test.expectedMeasurementEvent[i].Measurement,
					(<-actualMeasurementCh).Measurement,
					"Measurement is not expected",
				)
			}

			assert.Len(t, actualMeasurementCh, 0, "Unexpected count of measurement")
		})
	}
}
