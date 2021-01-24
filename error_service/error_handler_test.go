package error_service

import (
	"errors"
	"testing"
	"time"

	"github.com/pvelx/triggerhook/contracts"
	"github.com/stretchr/testify/assert"
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
	eventDebug := make(chan contracts.EventError, 1)
	eventError := make(chan contracts.EventError, 1)
	eventFatal := make(chan contracts.EventError, 1)
	err := make(chan error, 1)

	eventHandler := New(&Options{
		EventHandlers: map[contracts.Level]func(event contracts.EventError){
			contracts.LevelDebug: func(event contracts.EventError) { eventDebug <- event },
			contracts.LevelError: func(event contracts.EventError) { eventError <- event },
			contracts.LevelFatal: func(event contracts.EventError) { eventFatal <- event },
		},
		Debug:    true,
		EventCap: 0,
	})

	go func() {
		err <- eventHandler.Run()
	}()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			eventHandler.New(test.level, test.eventMessage, test.extra)
			var event contracts.EventError

			select {
			case event = <-eventDebug:
				assert.Equal(t, contracts.LevelDebug, event.Level, "must be level debug")
			case event = <-eventError:
				assert.Equal(t, contracts.LevelError, event.Level, "must be level error")
			case event = <-eventFatal:
				assert.Equal(t, contracts.LevelFatal, event.Level, "must be level fatal")
				assert.Equal(t, errors.New(test.eventMessage), <-err, "must be level fatal")
			case <-time.After(time.Second):
				assert.Fail(t, "does not receive error")
			}
		})
	}
}
