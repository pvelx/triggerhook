package services

import (
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain"
	"github.com/VladislavPav/trigger-hook/prioritized_task_list"
	"math"
	"sync"
	"time"
)

func NewWaitingTaskService(chPreloadedTask chan domain.Task, chTasksReadyToSend chan domain.Task) contracts.WaitingTaskServiceInterface {
	service := &waitingTaskService{
		tasksWaitingList:   prioritized_task_list.NewTaskQueueHeap([]domain.Task{}),
		chPreloadedTask:    chPreloadedTask,
		chTasksReadyToSend: chTasksReadyToSend,
		mu:                 &sync.Mutex{},
	}

	return service
}

type waitingTaskService struct {
	tasksWaitingList   contracts.PrioritizedTaskListInterface
	chPreloadedTask    chan domain.Task
	chTasksReadyToSend chan domain.Task
	mu                 *sync.Mutex
}

func (s *waitingTaskService) addTaskToWaitingList(task *domain.Task) {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	s.tasksWaitingList.Add(task)
}

func (s *waitingTaskService) takeTaskFromWaitingList() *domain.Task {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	return s.tasksWaitingList.Take()
}

func (s *waitingTaskService) WaitUntilExecTime() {
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
