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

func (s *preloadingTaskService) AddNewTask(tasks []*domain.Task) error {

	var taskToCreateContainer []contracts.TaskToCreate
	for _, task := range tasks {
		relativeTimeToExec := task.ExecTime - time.Now().Unix()

		taskToCreateContainer = append(
			taskToCreateContainer,
			contracts.TaskToCreate{Task: task, IsTaken: s.timePreload > relativeTimeToExec},
		)
	}

	if err := s.taskManager.Create(taskToCreateContainer); err != nil {
		return err
	}

	for _, taskToCreate := range taskToCreateContainer {
		if taskToCreate.IsTaken {
			s.chPreloadedTask <- *taskToCreate.Task
		}
	}

	return nil
}

func (s *preloadingTaskService) Preload() {
	countFails := 0

	for {
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

		workersCount := 10
		for worker := 0; worker < workersCount; worker++ {
			//workerNum := worker
			go func() {
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
			}()
		}
		time.Sleep(time.Hour)

	}
}
