package preloader_service

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"sync"
	"time"
)

func New(
	taskManager contracts.TaskManagerInterface,
	eventErrorHandler contracts.EventErrorHandlerInterface,
	monitoring contracts.MonitoringInterface,
) contracts.PreloadingTaskServiceInterface {

	//if err := monitoring.Init("", contracts.ValueMetricType); err != nil {
	//	panic(err)
	//}

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
		monitoring:               monitoring,
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
	monitoring               contracts.MonitoringInterface
}

func (s *preloadingTaskService) GetPreloadedChan() <-chan domain.Task {
	return s.chPreloadedTask
}

func (s *preloadingTaskService) AddNewTask(task *domain.Task) error {
	relativeTimeToExec := time.Duration(task.ExecTime-time.Now().Unix()) * time.Second
	isTaken := s.timePreload*time.Duration(s.coefTimePreloadOfNewTask) > relativeTimeToExec

	if err := s.taskManager.Create(task, isTaken); err != nil {
		return err
	}

	if isTaken {
		s.chPreloadedTask <- *task
	}

	return nil
}

func (s *preloadingTaskService) Preload() {
	for {
		result, err := s.taskManager.GetTasksToComplete(s.timePreload)
		switch {
		case err == contracts.TmErrorCollectionsNotFound:
			s.eh.New(contracts.LevelDebug, "I go to sleep because I don't get any tasks", nil)
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
		for worker := 0; worker < s.workersCount; worker++ {
			wg.Add(1)
			go s.getBunchOfTask(&wg, result, worker)
		}
		wg.Wait()
	}
}

func (s *preloadingTaskService) getBunchOfTask(wg *sync.WaitGroup, result contracts.CollectionsInterface, worker int) {
	defer wg.Done()
	for {
		tasks, err := result.Next()
		if err != nil {
			if err == contracts.NoCollections {
				s.eh.New(contracts.LevelDebug, "end", map[string]interface{}{
					"worker": worker,
				})
				break
			} else {
				/*
					Stop application
				*/
				s.eh.New(contracts.LevelFatal, "cannot get tasks for doing", map[string]interface{}{
					"worker": worker,
				})
				return
			}
		}

		s.eh.New(contracts.LevelDebug, "preloaded tasks", map[string]interface{}{
			"tasks count": len(tasks),
			"worker":      worker,
		})

		for _, task := range tasks {
			s.chPreloadedTask <- task
		}
	}
}
