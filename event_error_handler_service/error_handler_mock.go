package event_error_handler_service

import (
	"github.com/pvelx/triggerHook/contracts"
)

type ErrorHandlerMock struct {
	contracts.EventErrorHandlerInterface

	/*
		You need to substitute *Mock methods to do substitute original functions
	*/
	SetErrorHandlerMock func(level contracts.Level, eventHandler func(event contracts.EventError))
	NewMock             func(level contracts.Level, eventMessage string, extra map[string]interface{})
}

func (tm *ErrorHandlerMock) SetErrorHandler(level contracts.Level, eventHandler func(event contracts.EventError)) {
	if tm.SetErrorHandlerMock != nil {
		tm.SetErrorHandlerMock(level, eventHandler)
	}
}

func (tm *ErrorHandlerMock) New(level contracts.Level, eventMessage string, extra map[string]interface{}) {
	if tm.NewMock != nil {
		tm.NewMock(level, eventMessage, extra)
	}
}
