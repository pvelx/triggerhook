package error_service

import (
	"github.com/pvelx/triggerhook/contracts"
)

type ErrorHandlerMock struct {
	contracts.EventHandlerInterface

	/*
		You need to substitute *Mock methods to do substitute original functions
	*/
	NewMock func(level contracts.Level, eventMessage string, extra map[string]interface{})
}

func (tm *ErrorHandlerMock) New(level contracts.Level, eventMessage string, extra map[string]interface{}) {
	if tm.NewMock != nil {
		tm.NewMock(level, eventMessage, extra)
	}
}
