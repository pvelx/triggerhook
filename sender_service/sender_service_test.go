package sender_service

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/error_service"
	"github.com/pvelx/triggerHook/monitoring_service"
	"github.com/pvelx/triggerHook/task_manager"
	"github.com/pvelx/triggerHook/util"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func createTasks(chTaskReadyToSend chan domain.Task, count int) {
	for i := 0; i < count; i++ {
		chTaskReadyToSend <- domain.Task{Id: util.NewId(), ExecTime: time.Now().Unix()}
	}
}

/*
	Тестирование пакетных подтверждений в таск менеджере.

	Тест проверяет формирование пакетов задач на подтверждение.
	На размер пакета и периодичность отправки влияют праметры BatchMaxItems и BatchTimeout.
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
	chTaskReadyToSend := make(chan domain.Task, 100000)

	monitoringMock := &monitoring_service.MonitoringMock{
		InitMock:    func(topic contracts.Topic, metricType contracts.MetricType) error { return nil },
		PublishMock: func(topic contracts.Topic, measurement int64) error { return nil },
	}
	taskManagerMock := &task_manager.TaskManagerMock{ConfirmExecutionMock: func(tasks []domain.Task) error {
		assert.Equal(t, <-expTasksLen, len(tasks), "bunch of tasks has not correct length")
		time.Sleep(confirmationDuration)

		return nil
	}}

	senderService := New(taskManagerMock, chTaskReadyToSend, &error_service.ErrorHandlerMock{}, monitoringMock, &Options{
		BatchMaxItems: 1000,
		BatchTimeout:  50 * time.Millisecond,
	})

	go senderService.Run()

	go func() {
		for {
			senderService.Consume().Confirm()
		}
	}()

	go createTasks(chTaskReadyToSend, 200)
	time.Sleep(25 * time.Millisecond)
	go createTasks(chTaskReadyToSend, 200)
	time.Sleep(125 * time.Millisecond)
	go createTasks(chTaskReadyToSend, 1500)
	time.Sleep(75 * time.Millisecond)
	go createTasks(chTaskReadyToSend, 3300)

	//waiting for tasks to be processed
	time.Sleep(time.Second)

	assert.Len(t, expTasksLen, 0, "not all task was handle")
}

func TestConsuming(t *testing.T) {

	tries := 10
	countTasks := 6
	chTaskReadyToSend := make(chan domain.Task, countTasks)

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

	monitoringMock := &monitoring_service.MonitoringMock{
		InitMock:    func(topic contracts.Topic, metricType contracts.MetricType) error { return nil },
		PublishMock: func(topic contracts.Topic, measurement int64) error { return nil },
	}

	taskManagerMock := &task_manager.TaskManagerMock{ConfirmExecutionMock: func(tasks []domain.Task) error {
		assert.Len(t, tasks, countTasks, "count task is not correct")
		return nil
	}}

	senderService := New(taskManagerMock, chTaskReadyToSend, &error_service.ErrorHandlerMock{}, monitoringMock, nil)

	createTasks(chTaskReadyToSend, countTasks)
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
