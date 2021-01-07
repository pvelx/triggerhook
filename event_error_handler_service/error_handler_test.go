package event_error_handler_service

import (
	"errors"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSendEvent(t *testing.T) {
	tests := []struct {
		name         string
		level        contracts.Level
		eventMessage string
		extra        map[string]interface{}
	}{
		{
			name:         "debug level",
			level:        contracts.LevelDebug,
			eventMessage: "test level debug",
			extra: map[string]interface{}{
				"one":   1,
				"two":   "string",
				"three": int64(2),
				"fore":  int32(3),
			},
		},
		{
			name:         "error level",
			level:        contracts.LevelError,
			eventMessage: "test level error",
			extra: map[string]interface{}{
				"one":   1,
				"two":   "string",
				"three": int64(2),
				"fore":  int32(3),
			},
		},
		{
			name:         "error fatal",
			level:        contracts.LevelFatal,
			eventMessage: "test level error",
		},
	}

	eventErrorHandler := New(true)
	eventDebugCh := make(chan contracts.EventError, 1)
	eventErrorCh := make(chan contracts.EventError, 1)
	eventFatalCh := make(chan contracts.EventError, 1)
	errorCh := make(chan error, 1)

	eventErrorHandler.SetErrorHandler(contracts.LevelDebug, func(event contracts.EventError) {
		eventDebugCh <- event
	})
	eventErrorHandler.SetErrorHandler(contracts.LevelError, func(event contracts.EventError) {
		eventErrorCh <- event
	})
	eventErrorHandler.SetErrorHandler(contracts.LevelFatal, func(event contracts.EventError) {
		eventFatalCh <- event
	})

	go func() {
		errorCh <- eventErrorHandler.Run()
	}()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			eventErrorHandler.New(test.level, test.eventMessage, test.extra)
			var event contracts.EventError

			select {
			case event = <-eventDebugCh:
				assert.Equal(t, contracts.LevelDebug, event.Level, "must be level debug")
			case event = <-eventErrorCh:
				assert.Equal(t, contracts.LevelError, event.Level, "must be level error")
			case event = <-eventFatalCh:
				assert.Equal(t, contracts.LevelFatal, event.Level, "must be level fatal")
				assert.Equal(t, errors.New(test.eventMessage), <-errorCh, "must be level fatal")
			}
		})
	}
}
