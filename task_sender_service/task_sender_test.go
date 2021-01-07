package task_sender_service

import (
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/event_error_handler_service"
	"github.com/pvelx/triggerHook/task_manager"
	"github.com/pvelx/triggerHook/util"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func createReadyToSendTask(chTaskReadyToSend chan domain.Task, count int) {
	for i := 0; i < count; i++ {
		chTaskReadyToSend <- domain.Task{Id: util.NewId(), ExecTime: time.Now().Unix()}
	}
}

func TestTaskSender(t *testing.T) {
	mu := sync.Mutex{}
	expected := []int{400, 1000, 500, 1000, 1000, 1000, 300}

	chTaskReadyToSend := make(chan domain.Task, 10000000)
	taskManagerMock := &task_manager.TaskManagerMock{ConfirmExecutionMock: func(tasks []domain.Task) error {
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

	service := New(taskManagerMock, chTaskReadyToSend, nil, &event_error_handler_service.ErrorHandlerMock{}, nil)
	service.SetTransport(func(task domain.Task) {})

	go service.Run()

	go createReadyToSendTask(chTaskReadyToSend, 200)
	time.Sleep(25 * time.Millisecond)
	go createReadyToSendTask(chTaskReadyToSend, 200)
	time.Sleep(125 * time.Millisecond)
	go createReadyToSendTask(chTaskReadyToSend, 1500)
	time.Sleep(75 * time.Millisecond)
	go createReadyToSendTask(chTaskReadyToSend, 3300)
	time.Sleep(time.Second)
}
