package error_service

import (
	"errors"
	"github.com/pvelx/triggerhook/contracts"
	"sync"
	"time"
)

var EventQueueIsEmpty = errors.New("event queue is empty")
var SlowEventHandler = errors.New("event handler is very slow; some messages are lost")

type eventQueue struct {
	sync.Mutex
	maxQueueCap int
	queue       []contracts.EventError
}

func (t *eventQueue) Push(event contracts.EventError) {
	t.Lock()
	defer t.Unlock()

	if len(t.queue) == t.maxQueueCap {
		t.queue = t.queue[2:]
		t.queue = append(t.queue, contracts.EventError{
			Time:         time.Now(),
			Level:        contracts.LevelError,
			EventMessage: SlowEventHandler.Error(),
		})
	}

	t.queue = append(t.queue, event)
}

func (t *eventQueue) Pop() (event contracts.EventError, err error) {
	t.Lock()
	defer t.Unlock()

	if len(t.queue) > 0 {
		event, t.queue = t.queue[0], t.queue[1:]
		return
	}

	err = EventQueueIsEmpty
	return
}
