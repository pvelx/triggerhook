package waiting_service

import (
	"context"
	"math/rand"
	"sync/atomic"
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
	var inputCountOfTasks int32 = 10000
	dispersion := 5
	offset := 0
	pause := 5 * time.Second
	pauseStep := pause / time.Duration(inputCountOfTasks)
	preloadedTask := make(chan domain.Task, inputCountOfTasks)
	wait := make(chan bool)

	waitingService := instanceOfWaitingService(preloadedTask)

	go waitingService.Run()

	go func() {
		for i := int32(0); i < inputCountOfTasks; i++ {
			preloadedTask <- newTask(offset, dispersion)
			if i%(inputCountOfTasks/10) == 0 {
				time.Sleep(pauseStep)
			}
		}
		close(wait)
	}()

	var actualCountOfTasks int32
	go func() {
		for task := range waitingService.GetReadyToSendChan() {
			assert.Equal(t, time.Now().Unix(), task.ExecTime, "time execution in not equal to current time")
			atomic.AddInt32(&actualCountOfTasks, 1)
		}
	}()

	// wait for all tasks to be processed
	<-wait
	time.Sleep((time.Duration(offset + dispersion + 1)) * time.Second)

	assert.Equal(t, inputCountOfTasks, atomic.LoadInt32(&actualCountOfTasks), "tasks count is not correct")
}

func TestDeleteTask(t *testing.T) {
	inputCountOfTasks := 10000
	dispersion := 2
	offset := 2
	preloadedTask := make(chan domain.Task, inputCountOfTasks)
	var taskToDelete []domain.Task

	waitingService := instanceOfWaitingService(preloadedTask)

	go waitingService.Run()

	for i := 0; i < inputCountOfTasks; i++ {
		task := newTask(offset, dispersion)
		preloadedTask <- task
		taskToDelete = append(taskToDelete, task)
	}

	//need some time for process tasks
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()
	for _, task := range taskToDelete {
		if waitingService.CancelIfExist(ctx, task.Id) != nil {
			assert.Fail(t, "error is not expected")
		}
	}

	time.Sleep(time.Duration(offset+dispersion+1) * time.Second)

	assert.Equal(t, 0, len(waitingService.GetReadyToSendChan()), "tasks count is not correct")
}

func TestAddLateTask(t *testing.T) {
	var inputCountOfTasks int32 = 10000
	dispersion := 10
	offset := -10
	preloadedTask := make(chan domain.Task, inputCountOfTasks)

	waitingService := instanceOfWaitingService(preloadedTask)

	//run service
	go waitingService.Run()

	//receive task after waiting
	var actualCountOfTasks int32
	go func() {
		for task := range waitingService.GetReadyToSendChan() {
			now := time.Now().Unix()
			assert.Less(t, task.ExecTime, now+int64(offset+dispersion), "the task goes beyond the time")
			assert.GreaterOrEqual(t, task.ExecTime, now+int64(offset), "the task goes beyond the time")
			atomic.AddInt32(&actualCountOfTasks, 1)
		}
	}()

	for i := int32(0); i < inputCountOfTasks; i++ {
		preloadedTask <- newTask(offset, dispersion)
	}

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, inputCountOfTasks, atomic.LoadInt32(&actualCountOfTasks), "tasks count is not correct")
}

func instanceOfWaitingService(preloadedTask chan domain.Task) contracts.WaitingServiceInterface {
	return New(
		preloadedTask,
		&monitoring_service.MonitoringMock{},
		&task_manager.TaskManagerMock{DeleteMock: func(ctx context.Context, taskId string) error {
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
