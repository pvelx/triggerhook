package preloader_service

import (
	"sync"
	"time"

	"github.com/imdario/mergo"
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

type Options struct {
	TimePreload time.Duration

	/*
		Coefficient must be more than one. If the coefficient <= 1 then it may lead to save not taken task as taken
	*/
	CoefTimePreloadOfNewTask int
	TaskNumberInOneSearch    int
	WorkersCount             int
	PreloadedTaskCap         int //Deprecated
}

func New(
	taskManager contracts.TaskManagerInterface,
	eventHandler contracts.EventHandlerInterface,
	monitoring contracts.MonitoringInterface,
	options *Options,
) contracts.PreloadingServiceInterface {

	if options == nil {
		options = &Options{}
	}

	defaultOptions := Options{
		TimePreload:              5 * time.Second,
		CoefTimePreloadOfNewTask: 2,
		TaskNumberInOneSearch:    1000,
		WorkersCount:             10,
	}

	if err := mergo.Merge(options, defaultOptions); err != nil {
		panic(err)
	}

	preloadedTask := make(chan domain.Task, 1)

	if err := monitoring.Init(contracts.CreatingRate, contracts.VelocityMetricType); err != nil {
		panic(err)
	}
	if err := monitoring.Init(contracts.PreloadingRate, contracts.VelocityMetricType); err != nil {
		panic(err)
	}

	return &preloadingService{
		taskManager:              taskManager,
		eh:                       eventHandler,
		preloadedTask:            preloadedTask,
		timePreload:              options.TimePreload,
		coefTimePreloadOfNewTask: options.CoefTimePreloadOfNewTask,
		taskNumberInOneSearch:    options.TaskNumberInOneSearch,
		workersCount:             options.WorkersCount,
		monitoring:               monitoring,
	}
}

type preloadingService struct {
	taskManager              contracts.TaskManagerInterface
	eh                       contracts.EventHandlerInterface
	preloadedTask            chan domain.Task
	timePreload              time.Duration
	coefTimePreloadOfNewTask int
	taskNumberInOneSearch    int
	workersCount             int
	monitoring               contracts.MonitoringInterface
}

func (s *preloadingService) GetPreloadedChan() <-chan domain.Task {
	return s.preloadedTask
}

func (s *preloadingService) AddNewTask(task *domain.Task) error {
	relativeTimeToExec := time.Duration(task.ExecTime-time.Now().Unix()) * time.Second
	isTaken := s.timePreload*time.Duration(s.coefTimePreloadOfNewTask) > relativeTimeToExec

	if err := s.taskManager.Create(task, isTaken); err != nil {
		return err
	}

	if isTaken {
		s.preloadedTask <- *task
	}

	if err := s.monitoring.Publish(contracts.CreatingRate, 1); err != nil {
		s.eh.New(contracts.LevelError, err.Error(), nil)
	}

	return nil
}

func (s *preloadingService) Run() {
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

func (s *preloadingService) getBunchOfTask(wg *sync.WaitGroup, result contracts.CollectionsInterface, worker int) {
	defer wg.Done()
	for {
		tasks, err := result.Next()
		if err != nil {
			if err == contracts.RepoErrorNoCollections {
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

		if err := s.monitoring.Publish(contracts.PreloadingRate, int64(len(tasks))); err != nil {
			s.eh.New(contracts.LevelError, err.Error(), nil)
		}

		for _, task := range tasks {
			s.preloadedTask <- task
		}
	}
}
