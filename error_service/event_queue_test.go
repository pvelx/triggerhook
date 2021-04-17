package error_service

import (
	"github.com/pvelx/triggerhook/contracts"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEventQueue(t *testing.T) {
	expectedEventMessage := []string{
		"",
		"",
		"",
		"",
		SlowEventHandler.Error(),
		"",
		SlowEventHandler.Error(),
		"",
		SlowEventHandler.Error(),
		"",
	}

	maxQueueCap := 10
	eq := &eventQueue{
		maxQueueCap: maxQueueCap,
	}

	for i := 0; i < 13; i++ {
		eq.Push(contracts.EventError{})
	}

	for i := 0; ; i++ {
		e, err := eq.Pop()
		if err == EventQueueIsEmpty {
			break
		}
		assert.Equal(t, e.EventMessage, expectedEventMessage[i], "message of events are not equal")
	}

	assert.Equal(t, len(eq.queue), 0)
}
