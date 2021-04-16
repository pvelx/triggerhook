package error_service

import (
	"errors"
	"runtime"
	"time"

	"github.com/imdario/mergo"
	"github.com/pvelx/triggerhook/contracts"
)

var ErrorServiceClosed = errors.New("error service closed")

type Options struct {
	/*
		Configure the function using the desired error handler (for example, file logger, Sentry or other)
	*/
	EventHandlers map[contracts.Level]func(event contracts.EventError)
	Debug         bool
	EventCap      int
}

func New(options *Options) contracts.EventHandlerInterface {

	if options == nil {
		options = &Options{}
	}

	if err := mergo.Merge(options, Options{
		Debug:    false,
		EventCap: 1000,
	}); err != nil {
		panic(err)
	}

	return &EventHandler{
		updateQueue:   make(chan bool, 1),
		eventHandlers: options.EventHandlers,
		eventQueue:    &eventQueue{maxQueueCap: options.EventCap},
		debug:         options.Debug,
	}
}

type EventHandler struct {
	eventQueue    *eventQueue
	updateQueue   chan bool
	eventHandlers map[contracts.Level]func(event contracts.EventError)
	debug         bool
	contracts.EventHandlerInterface
}

func (eh *EventHandler) New(level contracts.Level, eventMessage string, extra map[string]interface{}) {

	if !eh.debug && level == contracts.LevelDebug {
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

	eh.eventQueue.Push(eventError)

	if len(eh.updateQueue) == 0 {
		eh.updateQueue <- true
	}
}

func (eh *EventHandler) Run() error {
	for {
		event, err := eh.eventQueue.Pop()
		if err != EventQueueIsEmpty {
			eventHandler, ok := eh.eventHandlers[event.Level]
			if !ok {
				continue
			}

			eventHandler(event)

			if event.Level == contracts.LevelFatal {
				return errors.New(event.EventMessage)
			}
		} else {
			if _, ok := <-eh.updateQueue; !ok {
				return ErrorServiceClosed
			}
		}
	}
}
