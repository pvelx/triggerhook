package error_service

import (
	"errors"
	"github.com/imdario/mergo"
	"github.com/pvelx/triggerHook/contracts"
	"runtime"
	"time"
)

type Options struct {
	/*
		Configure the function using the desired error handler (for example, file logger, Sentry or other)
	*/
	EventHandlers map[contracts.Level]func(event contracts.EventError)
	Debug         bool
	ChEventCap    int
}

func New(options *Options) contracts.EventErrorHandlerInterface {

	if options == nil {
		options = &Options{}
	}

	if err := mergo.Merge(options, Options{
		Debug:      false,
		ChEventCap: 1000000,
	}); err != nil {
		panic(err)
	}

	return &EventErrorHandler{
		chEvent:       make(chan contracts.EventError, options.ChEventCap),
		eventHandlers: options.EventHandlers,
		debug:         options.Debug,
	}
}

type EventErrorHandler struct {
	chEvent       chan contracts.EventError
	eventHandlers map[contracts.Level]func(event contracts.EventError)
	debug         bool
	contracts.EventErrorHandlerInterface
}

func (eeh *EventErrorHandler) New(level contracts.Level, eventMessage string, extra map[string]interface{}) {

	if !eeh.debug && level == contracts.LevelDebug {
		return
	}

	eventError := contracts.EventError{
		Time:         time.Now(),
		Level:        level,
		EventMessage: eventMessage,
		Extra:        extra,
	}

	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		file, line := details.FileLine(pc)
		eventError.Line = line
		eventError.File = file
		eventError.Method = details.Name()
	}

	eeh.chEvent <- eventError
}

func (eeh *EventErrorHandler) Run() error {
	for event := range eeh.chEvent {

		eventHandler, ok := eeh.eventHandlers[event.Level]
		if !ok {
			continue
		}

		eventHandler(event)

		if event.Level == contracts.LevelFatal {
			return errors.New(event.EventMessage)
		}
	}

	panic("channel of error handler was closed")
}
