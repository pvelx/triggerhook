package services

import (
	"errors"
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
		taskManager:       taskManager,
		eventErrorHandler: eventErrorHandler,
		chPreloadedTask:   make(chan domain.Task, 10000000),
		timePreload:       5 * time.Second,

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
	eventErrorHandler        contracts.EventErrorHandlerInterface
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
			fmt.Println(fmt.Sprintf("DEBUG: I am sleep %s because does not get any tasks", s.timePreload))
			time.Sleep(s.timePreload)
			continue
		case err != nil:
			/*
				Stop application
			*/
			s.eventErrorHandler.New(
				contracts.LevelFatal,
				errors.New("preloader cannot get bunches of tasks"),
				nil,
			)

			return
		}

		var wg sync.WaitGroup

		wg.Add(s.workersCount)
		for worker := 0; worker < s.workersCount; worker++ {
			go func() {
				for {
					tasks, isEnd, err := result.Next()
					if err != nil {
						panic("Cannot get tasks for doing")
						return
					}
					if isEnd {
						fmt.Println("DEBUG: end")
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
