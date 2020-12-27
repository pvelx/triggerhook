package services

import (
	"fmt"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/repository"
	"sync"
	"time"
)

func NewPreloadingTaskService(tm contracts.TaskManagerInterface) contracts.PreloadingTaskServiceInterface {
	return &preloadingTaskService{
		taskManager:           tm,
		chPreloadedTask:       make(chan domain.Task, 10000000),
		timePreload:           5 * time.Second,
		taskNumberInOneSearch: 1000,
	}
}

type preloadingTaskService struct {
	taskManager           contracts.TaskManagerInterface
	chPreloadedTask       chan domain.Task
	timePreload           time.Duration
	taskNumberInOneSearch int
}

func (s *preloadingTaskService) GetPreloadedChan() <-chan domain.Task {
	return s.chPreloadedTask
}

func (s *preloadingTaskService) AddNewTask(task domain.Task) error {
	relativeTimeToExec := time.Duration(task.ExecTime-time.Now().Unix()) * time.Second
	isTaken := s.timePreload > relativeTimeToExec

	if err := s.taskManager.Create(task, isTaken); err != nil {
		return err
	}

	if isTaken {
		s.chPreloadedTask <- task
	}

	return nil
}

func (s *preloadingTaskService) Preload() {
	countFails := 0

	for {
		result, err := s.taskManager.GetTasksToComplete(s.timePreload)
		switch {
		case err == repository.NoTasksFound:
			fmt.Printf("DEBUG: I am sleep %s because does not get any tasks", s.timePreload)
			time.Sleep(s.timePreload)
			continue
		case err != nil:
			countFails++
			time.Sleep(300 * time.Millisecond)
			if countFails == 10 {
				panic("Cannot get count of task for doing")
			}
			continue
		}
		countFails = 0

		var wg sync.WaitGroup
		workersCount := 10
		wg.Add(workersCount)
		for worker := 0; worker < workersCount; worker++ {
			//workerNum := worker
			go func() {
				for {
					tasks, isEnd, err := result.Next()
					if err != nil {
						panic("Cannot get tasks for doing")
						return
					}
					if isEnd {
						fmt.Printf("break")
						break
					}
					if len(tasks) == 0 {
						continue
					}
					//fmt.Printf("workerId: %d - %d", workerNum, len(tasks))
					//fmt.Println()

					for _, task := range tasks {
						s.chPreloadedTask <- task
					}
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}
