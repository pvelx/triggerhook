package services

import "github.com/pvelx/triggerHook/contracts"

type level int

const (
	//The application cannot continue
	LevelFatal level = iota

	//Must be delivered to support
	LevelError
)

type EventError struct {
	Level level
	Error error
}

func NewEventErrorHandler() contracts.EventErrorHandlerInterface {
	return &EventErrorHandler{chEventError: make(chan EventError, 1000)}
}

func (eeh *EventErrorHandler) NewEventError(level level, error error) {
	eeh.chEventError <- EventError{level, error}
}

type EventErrorHandler struct {
	chEventError         chan EventError
	externalErrorHandler func(event EventError)
	contracts.EventErrorHandlerInterface
}

func (eeh *EventErrorHandler) SetErrorHandler(externalErrorHandler func(event EventError)) {
	eeh.externalErrorHandler = externalErrorHandler
}

func (eeh *EventErrorHandler) Listen() error {
	for {
		select {
		case event := <-eeh.chEventError:
			eeh.externalErrorHandler(event)

			if event.Level == LevelFatal {
				return event.Error
			}
		}
	}
}
