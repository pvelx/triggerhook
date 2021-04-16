package sender_service

import (
	"sync"
	"testing"
	"time"

	"github.com/pvelx/triggerhook/domain"
	"github.com/pvelx/triggerhook/error_service"
	"github.com/pvelx/triggerhook/monitoring_service"
	"github.com/pvelx/triggerhook/task_manager"
	"github.com/stretchr/testify/assert"
)

func createTasks(taskReadyToSend chan domain.Task, count int) {
	for i := 0; i < count; i++ {
		taskReadyToSend <- domain.Task{ExecTime: time.Now().Unix()}
	}
}

func sum(s []int) (r int) {
	for _, v := range s {
		r += v
	}
	return
}

/*
	Testing batch confirmations in the task manager.

	The test checks the formation of task packages for confirmation.
	The batch size and frequency of sending are affected by the BatchMaxItems and BatchTimeout parameters.
*/
func TestConfirmation(t *testing.T) {
	inputTasksSequence := []struct {
		numberOfTask int
		timeout      time.Duration //timeout between tasks creation
	}{
		{200, 25 * time.Millisecond},
		{200, 125 * time.Millisecond},
		{1500, 75 * time.Millisecond},
		{3300, 0},
	}

	expectedTasksSequence := []int{400, 1000, 500, 1000, 1000, 1000, 300}

	//duration of saving in DB
	confirmationDuration := 100 * time.Millisecond

	taskReadyToSend := make(chan domain.Task, sum(expectedTasksSequence))

	done := make(chan struct{})
	mu := &sync.Mutex{}
	taskManagerMock := &task_manager.TaskManagerMock{ConfirmExecutionMock: func(tasks []domain.Task) error {
		var exp int
		mu.Lock()
		exp, expectedTasksSequence = expectedTasksSequence[0], expectedTasksSequence[1:]
		mu.Unlock()

		assert.Equal(t, exp, len(tasks), "bunch of tasks has not correct length")
		now := time.Now()

		//tasks is processed
		if len(expectedTasksSequence) == 0 {
			close(done)
		}

		//some work. Loop instead of sleep because tracing will better
		for time.Since(now) < confirmationDuration {
		}

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

	for _, v := range inputTasksSequence {
		createTasks(taskReadyToSend, v.numberOfTask)
		time.Sleep(v.timeout)
	}

	<-done

	assert.Len(t, expectedTasksSequence, 0, "not all task was handle")
}

func TestConsuming(t *testing.T) {

	expTries := 10
	countTasks := 6
	taskReadyToSend := make(chan domain.Task, countTasks)

	success := make(chan bool, expTries)
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

	done := make(chan struct{})
	go func() {
		for {
			result := senderService.Consume()

			if <-success {
				result.Confirm()
			} else {
				result.Rollback()
			}

			actualTries++

			if (actualTries) == expTries {
				close(done)
			}
		}
	}()

	<-done

	assert.Equal(t, expTries, actualTries, "tries count is not correct")
}
