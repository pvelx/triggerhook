package sender_service

import (
	"testing"
	"time"

	"github.com/pvelx/triggerhook/domain"
	"github.com/pvelx/triggerhook/error_service"
	"github.com/pvelx/triggerhook/monitoring_service"
	"github.com/pvelx/triggerhook/task_manager"
	"github.com/pvelx/triggerhook/util"
	"github.com/stretchr/testify/assert"
)

func createTasks(taskReadyToSend chan domain.Task, count int) {
	for i := 0; i < count; i++ {
		taskReadyToSend <- domain.Task{Id: util.NewId(), ExecTime: time.Now().Unix()}
	}
}

/*
	Testing batch confirmations in the task manager.

	The test checks the formation of task packages for confirmation.
	The batch size and frequency of sending are affected by the BatchMaxItems and BatchTimeout parameters.
*/
func TestConfirmation(t *testing.T) {
	expTasksLen := make(chan int, 7)
	expTasksLen <- 400
	expTasksLen <- 1000
	expTasksLen <- 500
	expTasksLen <- 1000
	expTasksLen <- 1000
	expTasksLen <- 1000
	expTasksLen <- 300

	confirmationDuration := 100 * time.Millisecond
	taskReadyToSend := make(chan domain.Task, 100000)

	taskManagerMock := &task_manager.TaskManagerMock{ConfirmExecutionMock: func(tasks []domain.Task) error {
		assert.Equal(t, <-expTasksLen, len(tasks), "bunch of tasks has not correct length")
		time.Sleep(confirmationDuration)

		return nil
	}}

	senderService := New(taskManagerMock, taskReadyToSend, &error_service.ErrorHandlerMock{}, &monitoring_service.MonitoringMock{}, &Options{
		BatchMaxItems: 1000,
		BatchTimeout:  50 * time.Millisecond,
	})

	go senderService.Run()

	go func() {
		for {
			senderService.Consume().Confirm()
		}
	}()

	go createTasks(taskReadyToSend, 200)
	time.Sleep(25 * time.Millisecond)
	go createTasks(taskReadyToSend, 200)
	time.Sleep(125 * time.Millisecond)
	go createTasks(taskReadyToSend, 1500)
	time.Sleep(75 * time.Millisecond)
	go createTasks(taskReadyToSend, 3300)

	//waiting for tasks to be processed
	time.Sleep(time.Second)

	assert.Len(t, expTasksLen, 0, "not all task was handle")
}

func TestConsuming(t *testing.T) {

	tries := 10
	countTasks := 6
	taskReadyToSend := make(chan domain.Task, countTasks)

	success := make(chan bool, tries)
	success <- true
	success <- true
	success <- true
	success <- false
	success <- false
	success <- true
	success <- false
	success <- true
	success <- false
	success <- true

	taskManagerMock := &task_manager.TaskManagerMock{ConfirmExecutionMock: func(tasks []domain.Task) error {
		assert.Len(t, tasks, countTasks, "count task is not correct")
		return nil
	}}

	senderService := New(
		taskManagerMock,
		taskReadyToSend,
		&error_service.ErrorHandlerMock{},
		&monitoring_service.MonitoringMock{},
		nil,
	)

	createTasks(taskReadyToSend, countTasks)
	go senderService.Run()

	actualTries := 0

	go func() {
		for {
			result := senderService.Consume()

			if <-success {
				result.Confirm()
			} else {
				result.Rollback()
			}

			actualTries++
		}
	}()

	//waiting for tasks to be processed
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, tries, actualTries, "tries count is not correct")
}
