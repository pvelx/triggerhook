package services

import "github.com/pvelx/triggerHook/contracts"

func NewEventErrorHandler() contracts.EventErrorHandlerInterface {
	return &EventErrorHandler{chEventError: make(chan contracts.EventError, 10000000)}
}

func (eeh *EventErrorHandler) NewEventError(level contracts.Level, error error) {
	eeh.chEventError <- contracts.EventError{Level: level, Error: error}
}

type EventErrorHandler struct {
	chEventError         chan contracts.EventError
	externalErrorHandler func(event contracts.EventError)
	contracts.EventErrorHandlerInterface
}

func (eeh *EventErrorHandler) SetErrorHandler(externalErrorHandler func(event contracts.EventError)) {
	eeh.externalErrorHandler = externalErrorHandler
}

func (eeh *EventErrorHandler) Listen() error {
	for {
		select {
		case event := <-eeh.chEventError:
			eeh.externalErrorHandler(event)

			if event.Level == contracts.LevelFatal {
				return event.Error
			}
		}
	}
}
