package services

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/prioritized_task_list"
	"math"
	"sync"
	"time"
)

func NewWaitingTaskService(chPreloadedTasks <-chan domain.Task) contracts.WaitingTaskServiceInterface {
	service := &waitingTaskService{
		tasksWaitingList:   prioritized_task_list.NewHeapPrioritizedTaskList([]domain.Task{}),
		chPreloadedTasks:   chPreloadedTasks,
		chCanceledTasks:    make(chan string, 10000000),
		chTasksReadyToSend: make(chan domain.Task, 10000000),
		mu:                 &sync.Mutex{},
	}

	return service
}

type waitingTaskService struct {
	tasksWaitingList   contracts.PrioritizedTaskListInterface
	chPreloadedTasks   <-chan domain.Task
	chCanceledTasks    chan string
	chTasksReadyToSend chan domain.Task
	mu                 *sync.Mutex
}

func (s *waitingTaskService) GetReadyToSendChan() <-chan domain.Task {
	return s.chTasksReadyToSend
}

func (s *waitingTaskService) deleteTaskFromWaitingList(taskId string) {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	s.tasksWaitingList.DeleteIfExist(taskId)
}

func (s *waitingTaskService) addTaskToWaitingList(task domain.Task) {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	s.tasksWaitingList.Add(task)
}

func (s *waitingTaskService) takeTaskFromWaitingList() *domain.Task {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	return s.tasksWaitingList.Take()
}

func (s *waitingTaskService) CancelIfExist(taskId string) {
	s.chCanceledTasks <- taskId
}

func (s *waitingTaskService) WaitUntilExecTime() {
	updatedQueue := make(chan bool)

	go func() {
		for {
			select {
			case taskId, ok := <-s.chCanceledTasks:
				if ok {
					s.deleteTaskFromWaitingList(taskId)
					updatedQueue <- true
				} else {
					panic("chan was closed")
				}
			case task, ok := <-s.chPreloadedTasks:
				if ok {
					s.addTaskToWaitingList(task)
					updatedQueue <- true
				} else {
					panic("chan was closed")
				}
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
					s.addTaskToWaitingList(*task)
				}

				continue
			}
		}

		s.chTasksReadyToSend <- *task
	}
}
