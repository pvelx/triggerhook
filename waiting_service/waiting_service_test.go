package waiting_service

import (
	"math/rand"
	"testing"
	"time"

	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
	"github.com/pvelx/triggerhook/monitoring_service"
	"github.com/pvelx/triggerhook/task_manager"
	"github.com/pvelx/triggerhook/util"
	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestAddNormalTask(t *testing.T) {
	inputCountOfTasks := 10000
	dispersion := 5
	offset := 0
	pause := 5 * time.Second
	pauseStep := pause / time.Duration(inputCountOfTasks)
	chPreloadedTask := make(chan domain.Task, inputCountOfTasks)
	wait := make(chan bool)

	waitingService := instanceOfWaitingService(chPreloadedTask)

	go waitingService.Run()

	go func() {
		for i := 0; i < inputCountOfTasks; i++ {
			chPreloadedTask <- newTask(offset, dispersion)
			if i%(inputCountOfTasks/10) == 0 {
				time.Sleep(pauseStep)
			}
		}
		close(wait)
	}()

	actualCountOfTasks := 0
	go func() {
		for task := range waitingService.GetReadyToSendChan() {
			assert.Equal(t, time.Now().Unix(), task.ExecTime, "time execution in not equal to current time")
			actualCountOfTasks++
		}
	}()

	// wait for all tasks to be processed
	<-wait
	time.Sleep((time.Duration(offset + dispersion + 1)) * time.Second)

	assert.Equal(t, inputCountOfTasks, actualCountOfTasks, "tasks count is not correct")
}

func TestDeleteTask(t *testing.T) {
	inputCountOfTasks := 10000
	dispersion := 2
	offset := 2
	chPreloadedTask := make(chan domain.Task, inputCountOfTasks)
	var taskToDelete []domain.Task

	waitingService := instanceOfWaitingService(chPreloadedTask)

	go waitingService.Run()

	for i := 0; i < inputCountOfTasks; i++ {
		task := newTask(offset, dispersion)
		chPreloadedTask <- task
		taskToDelete = append(taskToDelete, task)
	}

	//need some time for process tasks
	time.Sleep(100 * time.Millisecond)

	for _, task := range taskToDelete {
		if waitingService.CancelIfExist(task.Id) != nil {
			assert.Fail(t, "error is not expected")
		}
	}

	time.Sleep(time.Duration(offset+dispersion+1) * time.Second)

	assert.Equal(t, 0, len(waitingService.GetReadyToSendChan()), "tasks count is not correct")
}

func TestAddLateTask(t *testing.T) {
	inputCountOfTasks := 10000
	dispersion := 10
	offset := -10
	chPreloadedTask := make(chan domain.Task, inputCountOfTasks)

	waitingService := instanceOfWaitingService(chPreloadedTask)

	//run service
	go waitingService.Run()

	//receive task after waiting
	actualCountOfTasks := 0
	go func() {
		for task := range waitingService.GetReadyToSendChan() {
			now := time.Now().Unix()
			assert.Less(t, task.ExecTime, now+int64(offset+dispersion), "the task goes beyond the time")
			assert.GreaterOrEqual(t, task.ExecTime, now+int64(offset), "the task goes beyond the time")
			actualCountOfTasks++
		}
	}()

	for i := 0; i < inputCountOfTasks; i++ {
		chPreloadedTask <- newTask(offset, dispersion)
	}

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, inputCountOfTasks, actualCountOfTasks, "tasks count is not correct")
}

func instanceOfWaitingService(chPreloadedTask chan domain.Task) contracts.WaitingServiceInterface {
	return New(
		chPreloadedTask,
		&monitoring_service.MonitoringMock{},
		&task_manager.TaskManagerMock{DeleteMock: func(taskId string) error {
			return nil
		}},
		nil,
		nil,
	)
}

func newTask(offset, dispersion int) domain.Task {
	return domain.Task{
		Id:       util.NewId(),
		ExecTime: time.Now().Add(time.Duration(offset+rand.Intn(dispersion)) * time.Second).Unix(),
	}
}
