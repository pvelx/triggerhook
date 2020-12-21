package services

import (
	"fmt"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

func NewPreloadingTaskService(tm contracts.TaskManagerInterface) contracts.PreloadingTaskServiceInterface {
	return &preloadingTaskService{
		taskManager:           tm,
		chPreloadedTask:       make(chan domain.Task, 10000000),
		timePreload:           5,
		taskNumberInOneSearch: 1000,
	}
}

type preloadingTaskService struct {
	taskManager           contracts.TaskManagerInterface
	chPreloadedTask       chan domain.Task
	timePreload           int64
	taskNumberInOneSearch int
}

func (s *preloadingTaskService) GetPreloadedChan() <-chan domain.Task {
	return s.chPreloadedTask
}

func (s *preloadingTaskService) AddNewTask(execTime int64) (*domain.Task, error) {
	task := domain.Task{
		ExecTime: execTime,
	}
	relativeTimeToExec := execTime - time.Now().Unix()

	isTaken := s.timePreload > relativeTimeToExec

	if err := s.taskManager.Create([]contracts.TaskToCreate{{&task, isTaken}}); err != nil {
		return nil, err
	}
	if isTaken {
		s.chPreloadedTask <- task
	}
	return &task, nil
}

func (s *preloadingTaskService) Preload() {
	countFails := 0

	for {
		collection, err := s.taskManager.GetTasksBySecToExecTime(s.timePreload)
		if err != nil {
			countFails++
			time.Sleep(time.Duration(s.timePreload) * 300 * time.Millisecond)

			if countFails == 10 {
				panic("Cannot get count of task for doing")
			}
			continue
		}
		countFails = 0

		workersCount := 5
		for worker := 0; worker < workersCount; worker++ {
			//workerNum := worker
			go func() {
				for {
					tasksToExec, err := collection.Next()

					if len(tasksToExec) == 0 {
						fmt.Printf("break")
						continue
					}
					//fmt.Printf("workerId: %d - %d", workerNum, len(tasksToExec))
					//fmt.Println()

					if err != nil {
						panic("Cannot get tasks for doing")
						return
					}
					for _, task := range tasksToExec {
						s.chPreloadedTask <- task
					}
				}
			}()
		}
		time.Sleep(time.Hour)

	}
}
