package services

import (
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"github.com/VladislavPav/trigger-hook/services/structures/task_queue_heap"
	"math"
	"sync"
	"time"
)

type TaskHandlerServiceInterface interface {
	WaitUntilExecTime()
}

func NewTaskHandlerService(chPreloadedTask chan tasks.Task, chTasksReadyToSend chan tasks.Task) TaskHandlerServiceInterface {
	service := &taskHandlerService{
		tasksWaitingList:   task_queue_heap.NewTaskQueueHeap([]tasks.Task{}),
		chPreloadedTask:    chPreloadedTask,
		chTasksReadyToSend: chTasksReadyToSend,
		mu:                 &sync.Mutex{},
	}

	return service
}

type taskHandlerService struct {
	tasksWaitingList   contracts.TasksWaitingListInterface
	chPreloadedTask    chan tasks.Task
	chTasksReadyToSend chan tasks.Task
	mu                 *sync.Mutex
}

func (s *taskHandlerService) addTaskToWaitingList(task *tasks.Task) {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	s.tasksWaitingList.Add(task)
}

func (s *taskHandlerService) takeTaskFromWaitingList() *tasks.Task {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	return s.tasksWaitingList.Take()
}

func (s *taskHandlerService) WaitUntilExecTime() {
	updatedQueue := make(chan bool)

	go func() {
		for {
			select {
			case task := <-s.chPreloadedTask:
				s.addTaskToWaitingList(&task)
				updatedQueue <- true
			}
		}
	}()

	var sleepTime int64

	for {
		task := s.takeTaskFromWaitingList()

		if task == nil {
			sleepTime = math.MaxInt64
		} else {
			sleepTime = (task.ExecTime * 1e+9) - time.Now().UnixNano()
		}

		if sleepTime > 0 {
			timer := time.NewTimer(time.Duration(sleepTime) * time.Nanosecond)
			for len(updatedQueue) > 0 {
				<-updatedQueue
			}
			select {
			case <-timer.C:
				break
			case <-updatedQueue:
				if !timer.Stop() {
					<-timer.C
				}
				if task != nil {
					s.addTaskToWaitingList(task)
				}

				continue
			}
		}

		s.chTasksReadyToSend <- *task
	}
}
