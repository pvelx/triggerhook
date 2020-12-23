package services

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

type taskManagerMock struct {
	contracts.TaskManagerInterface
	confirmExecutionMock func(tasks []*domain.Task) error
}

func (tm *taskManagerMock) ConfirmExecution(tasks []*domain.Task) error {
	return tm.confirmExecutionMock(tasks)
}

func createReadyToSendTask(chTaskReadyToSend chan domain.Task, count int) {
	for i := 0; i < count; i++ {
		chTaskReadyToSend <- domain.Task{}
	}
}

func TestTaskSender(t *testing.T) {
	mu := sync.Mutex{}
	expected := []int{400, 1000, 500, 1000, 1000, 1000, 300}

	chTaskReadyToSend := make(chan domain.Task, 10000000)
	taskManagerMock := &taskManagerMock{confirmExecutionMock: func(tasks []*domain.Task) error {
		mu.Lock()
		if len(expected) == 0 {
			assert.Fail(t, "expected values ware end")
			return nil
		}
		expectedLen := expected[0]
		expected = expected[1:]
		mu.Unlock()
		assert.Equal(t, expectedLen, len(tasks), "bunch of tasks has not correct length")
		time.Sleep(100 * time.Millisecond)
		return nil
	}}

	service := NewTaskSender(taskManagerMock, chTaskReadyToSend, nil)
	service.SetTransport(func(task *domain.Task) {})

	go service.Send()

	go createReadyToSendTask(chTaskReadyToSend, 200)
	time.Sleep(25 * time.Millisecond)
	go createReadyToSendTask(chTaskReadyToSend, 200)
	time.Sleep(125 * time.Millisecond)
	go createReadyToSendTask(chTaskReadyToSend, 1500)
	time.Sleep(75 * time.Millisecond)
	go createReadyToSendTask(chTaskReadyToSend, 3300)
	time.Sleep(time.Second)
}
