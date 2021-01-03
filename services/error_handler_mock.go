package services

import (
	"github.com/pvelx/triggerHook/contracts"
)

type ErrorHandlerMock struct {
	contracts.EventErrorHandlerInterface

	/*
		You need to substitute *Mock methods to do substitute original functions
	*/
	setErrorHandlerMock func(level contracts.Level, eventHandler func(event contracts.EventError))
	newMock             func(level contracts.Level, eventMessage string, extra map[string]interface{})
}

func (tm *ErrorHandlerMock) SetErrorHandler(level contracts.Level, eventHandler func(event contracts.EventError)) {
	tm.setErrorHandlerMock(level, eventHandler)
}

func (tm *ErrorHandlerMock) New(level contracts.Level, eventMessage string, extra map[string]interface{}) {
	tm.newMock(level, eventMessage, extra)
}
