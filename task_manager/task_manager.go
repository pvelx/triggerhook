package task_manager

import (
	"context"
	"time"

	"github.com/imdario/mergo"
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
	"github.com/pvelx/triggerhook/util"
)

type Options struct {
	MaxRetry            int
	TimeGapBetweenRetry time.Duration
}

func New(
	repository contracts.RepositoryInterface,
	eh contracts.EventHandlerInterface,
	monitoring contracts.MonitoringInterface,
	options *Options,
) contracts.TaskManagerInterface {

	if err := repository.Up(); err != nil {
		panic(err)
	}

	if options == nil {
		options = &Options{}
	}

	if err := mergo.Merge(options, Options{
		MaxRetry:            3,
		TimeGapBetweenRetry: 10 * time.Millisecond,
	}); err != nil {
		panic(err)
	}

	count, err := repository.Count()
	if err != nil {
		panic(err)
	}

	if err := monitoring.Init(contracts.All, contracts.IntegralMetricType); err != nil {
		panic(err)
	}
	if err := monitoring.Publish(contracts.All, int64(count)); err != nil {
		panic(err)
	}

	return &taskManager{
		repository:          repository,
		eh:                  eh,
		maxRetry:            options.MaxRetry,
		timeGapBetweenRetry: options.TimeGapBetweenRetry,
		monitoring:          monitoring,
	}
}

type taskManager struct {
	contracts.TaskManagerInterface
	repository          contracts.RepositoryInterface
	eh                  contracts.EventHandlerInterface
	maxRetry            int
	timeGapBetweenRetry time.Duration
	monitoring          contracts.MonitoringInterface
}

func (s *taskManager) Create(ctx context.Context, task *domain.Task, isTaken bool) error {
	if now := time.Now().Unix(); task.ExecTime < now {
		task.ExecTime = now
	}

	if task.Id == "" {
		task.Id = util.NewId()

	} else if !util.IsIdValid(task.Id) {
		return contracts.TmErrorUuidIsNotCorrect
	}

	err := s.retry(func() error {
		return s.repository.Create(ctx, *task, isTaken)
	}, contracts.RepoErrorDeadlock)

	if err == contracts.RepoErrorTaskExist {
		s.eh.New(contracts.LevelDebug, err.Error(), map[string]interface{}{
			"task": task,
		})

		return contracts.TmErrorTaskExist
	} else if err != nil {
		s.eh.New(contracts.LevelError, err.Error(), map[string]interface{}{
			"task": task,
		})

		return contracts.TmErrorCreatingTasks
	}

	if err := s.monitoring.Publish(contracts.All, 1); err != nil {
		s.eh.New(contracts.LevelError, err.Error(), nil)
	}

	return nil
}

func (s *taskManager) Delete(ctx context.Context, taskId string) error {
	var affected int64
	errDeleting := s.retry(func() (err error) {
		affected, err = s.repository.Delete(ctx, []domain.Task{{Id: taskId}})
		return
	}, contracts.RepoErrorDeadlock)

	if errDeleting != nil {
		s.eh.New(contracts.LevelError, errDeleting.Error(), map[string]interface{}{
			"taskId": taskId,
		})

		return contracts.TmErrorDeletingTask
	}

	if affected == 0 {
		return contracts.TmErrorTaskNotFound
	}

	if err := s.monitoring.Publish(contracts.All, -affected); err != nil {
		s.eh.New(contracts.LevelError, err.Error(), nil)
	}

	return nil
}

func (s *taskManager) GetTasksToComplete(ctx context.Context, preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error) {
	var collections contracts.CollectionsInterface
	errFinding := s.retry(func() (err error) {
		collections, err = s.repository.FindBySecToExecTime(ctx, preloadingTimeRange)
		return
	}, contracts.RepoErrorDeadlock, contracts.RepoErrorLockWaitTimeout)

	switch {
	case errFinding == contracts.RepoErrorNoTasksFound:
		return nil, contracts.TmErrorCollectionsNotFound
	case errFinding != nil:
		s.eh.New(contracts.LevelError, errFinding.Error(), nil)
		return collections, contracts.TmErrorGetTasks
	}

	return collections, nil
}

func (s *taskManager) ConfirmExecution(ctx context.Context, tasks []domain.Task) error {
	var affected int64
	errConfirm := s.retry(func() (err error) {
		affected, err = s.repository.Delete(ctx, tasks)
		return
	}, contracts.RepoErrorDeadlock)

	if errConfirm != nil {
		s.eh.New(contracts.LevelError, errConfirm.Error(), map[string]interface{}{
			"count of task": len(tasks),
		})
		return contracts.TmErrorConfirmationTasks
	}

	if err := s.monitoring.Publish(contracts.All, -affected); err != nil {
		s.eh.New(contracts.LevelError, err.Error(), nil)
	}

	return nil
}

func (s *taskManager) retry(callback func() error, retryableErrors ...error) (err error) {
	for try := 1; try <= s.maxRetry; try++ {
		if err = callback(); err != nil {
			if util.Contains(retryableErrors, err) {
				s.eh.New(contracts.LevelError, err.Error(), map[string]interface{}{
					"try": try,
				})
				if try != s.maxRetry {
					time.Sleep(s.timeGapBetweenRetry)
				}

				continue
			}
		}

		break
	}

	return
}
