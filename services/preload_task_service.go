package services

import (
	"fmt"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"sync"
	"time"
)

func NewPreloadingTaskService(
	taskManager contracts.TaskManagerInterface,
	eventErrorHandler contracts.EventErrorHandlerInterface,
) contracts.PreloadingTaskServiceInterface {
	return &preloadingTaskService{
		taskManager:     taskManager,
		eh:              eventErrorHandler,
		chPreloadedTask: make(chan domain.Task, 10000000),
		timePreload:     5 * time.Second,

		/*
			Coefficient must be more than one. If the coefficient <= 1 then it may lead to save not taken task as taken
		*/
		coefTimePreloadOfNewTask: 2,
		taskNumberInOneSearch:    1000,
		workersCount:             10,
	}
}

type preloadingTaskService struct {
	taskManager              contracts.TaskManagerInterface
	eh                       contracts.EventErrorHandlerInterface
	chPreloadedTask          chan domain.Task
	timePreload              time.Duration
	coefTimePreloadOfNewTask int
	taskNumberInOneSearch    int
	workersCount             int
}

func (s *preloadingTaskService) GetPreloadedChan() <-chan domain.Task {
	return s.chPreloadedTask
}

func (s *preloadingTaskService) AddNewTask(task domain.Task) error {
	relativeTimeToExec := time.Duration(task.ExecTime-time.Now().Unix()) * time.Second
	isTaken := s.timePreload*time.Duration(s.coefTimePreloadOfNewTask) > relativeTimeToExec

	if err := s.taskManager.Create(task, isTaken); err != nil {
		return err
	}

	if isTaken {
		s.chPreloadedTask <- task
	}

	return nil
}

func (s *preloadingTaskService) Preload() {
	for {
		result, err := s.taskManager.GetTasksToComplete(s.timePreload)
		switch {
		case err == contracts.NoTasksFound:
			s.eh.New(contracts.LevelDebug, fmt.Sprintf("I go to sleep because I don't get any tasks"), nil)
			time.Sleep(s.timePreload)
			continue
		case err != nil:
			/*
				Stop application
			*/
			s.eh.New(contracts.LevelFatal, "preloader cannot get bunches of tasks", nil)

			return
		}

		var wg sync.WaitGroup
		wg.Add(s.workersCount)
		for worker := 0; worker < s.workersCount; worker++ {
			go func() {
				for {
					tasks, isEnd, err := result.Next()
					s.eh.New(contracts.LevelDebug, fmt.Sprintf("preloaded tasks"), map[string]interface{}{
						"tasks count": len(tasks),
					})
					if err != nil {
						s.eh.New(contracts.LevelFatal, fmt.Sprintf("cannot get tasks for doing"), nil)
						return
					}
					if isEnd {
						s.eh.New(contracts.LevelDebug, fmt.Sprintf("end"), nil)
						break
					}
					if len(tasks) == 0 {
						continue
					}

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
