package services

import (
	"fmt"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"sync"
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

func (s *preloadingTaskService) AddNewTask(task domain.Task) error {
	relativeTimeToExec := task.ExecTime - time.Now().Unix()
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
		now := time.Now()
		collection, err := s.taskManager.GetTasksToComplete(s.timePreload)
		if err != nil {
			countFails++
			time.Sleep(time.Duration(s.timePreload) * 300 * time.Millisecond)

			if countFails == 10 {
				panic("Cannot get count of task for doing")
			}
			continue
		}
		countFails = 0

		var wg sync.WaitGroup
		workersCount := 10
		for worker := 0; worker < workersCount; worker++ {
			//workerNum := worker
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				for {
					tasks, isEnd, err := collection.Next()
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
			}(&wg)
		}
		wg.Wait()
		time.Sleep(time.Duration((5 - time.Since(now).Seconds()) * float64(time.Second)))
	}
}
