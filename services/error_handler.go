package services

import (
	"errors"
	"fmt"
	"github.com/pvelx/triggerHook/contracts"
	"path"
	"runtime"
	"time"
)

var (
	baseFormat = "%s MESSAGE:%s METHOD:%s FILE:%s:%d EXTRA:%v\n"
	formats    = map[contracts.Level]string{
		contracts.LevelDebug: "DEBUG:" + baseFormat,
		contracts.LevelError: "ERROR:" + baseFormat,
		contracts.LevelFatal: "FATAL:" + baseFormat,
	}
)

func NewEventErrorHandler(debug bool) contracts.EventErrorHandlerInterface {
	eventHandlers := make(map[contracts.Level]func(event contracts.EventError))
	for level, format := range formats {
		format := format
		level := level
		eventHandlers[level] = func(event contracts.EventError) {
			_, shortMethod := path.Split(event.Method)
			_, shortFile := path.Split(event.File)
			fmt.Printf(
				format,
				event.Time.Format("2006-01-02 15:04:05.000"),
				event.EventMessage,
				shortMethod,
				shortFile,
				event.Line,
				event.Extra,
			)
		}
	}

	return &EventErrorHandler{
		chEvent:       make(chan contracts.EventError, 10000000),
		eventHandlers: eventHandlers,
		debug:         debug,
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

func (eeh *EventErrorHandler) SetErrorHandler(level contracts.Level, eventHandler func(event contracts.EventError)) {
	eeh.eventHandlers[level] = eventHandler
}

func (eeh *EventErrorHandler) Listen() error {
	for {
		select {
		case event := <-eeh.chEvent:
			eventHandler, ok := eeh.eventHandlers[event.Level]
			if !ok {
				return errors.New(fmt.Sprintf("event handler for level:%d must be defined", event.Level))
			}

			eventHandler(event)

			if event.Level == contracts.LevelFatal {
				return errors.New(event.EventMessage)
			}
		}
	}
}
