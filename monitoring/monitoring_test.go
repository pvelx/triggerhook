package monitoring

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMainFlow(t *testing.T) {
	tests := []struct {
		name            string
		inputMeasure    []int64
		expectedMeasure []int64
		periodInputPub  time.Duration
		periodSending   time.Duration
		metricType      contracts.MetricType
	}{
		{
			name:            "absolute metric",
			inputMeasure:    []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			expectedMeasure: []int64{1, 3, 5, 7, 9},
			periodInputPub:  50 * time.Millisecond,
			periodSending:   100 * time.Millisecond,
			metricType:      contracts.Absolute,
		},
		{
			name:            "periodic metric",
			inputMeasure:    []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			expectedMeasure: []int64{2, 10, 18, 26, 34},
			periodInputPub:  50 * time.Millisecond,
			periodSending:   100 * time.Millisecond,
			metricType:      contracts.Periodic,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			topicName := "oneTopic"
			monitoringService := NewMonitoringService(test.periodSending)
			monitoringService.Init(topicName, test.metricType)
			go monitoringService.Run()
			time.Sleep(10 * time.Millisecond)

			var actualMeasureCh = make(chan int64)
			consumer := func(measure int64) {
				actualMeasureCh <- measure
			}

			if _, err := monitoringService.Sub(topicName, consumer); err != nil {
				t.Fatal(err)
			}

			blockCh := make(chan bool)
			for worker := 0; worker < 2; worker++ {
				go func() {
					<-blockCh
					for _, measure := range test.inputMeasure {
						if err := monitoringService.Pub(topicName, measure); err != nil {
							t.Fatal(err)
						}
						time.Sleep(test.periodInputPub)
					}
				}()
			}
			close(blockCh)

			current := 0
			for actualMeasure := range actualMeasureCh {
				assert.Equal(t, test.expectedMeasure[current], actualMeasure, "Measure is not expected")
				current++
				if len(test.expectedMeasure) == current {
					break
				}
			}
		})
	}
}
