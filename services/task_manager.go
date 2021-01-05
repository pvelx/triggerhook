package services

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/util"
	"time"
)

func NewTaskManager(
	repository contracts.RepositoryInterface,
	eeh contracts.EventErrorHandlerInterface,
) contracts.TaskManagerInterface {
	return &taskManager{
		repository:          repository,
		eeh:                 eeh,
		maxRetry:            3,
		timeGapBetweenRetry: 10 * time.Millisecond,
	}
}

type taskManager struct {
	contracts.TaskManagerInterface
	repository          contracts.RepositoryInterface
	eeh                 contracts.EventErrorHandlerInterface
	maxRetry            int
	timeGapBetweenRetry time.Duration
}

func (s *taskManager) Create(task *domain.Task, isTaken bool) error {
	if now := time.Now().Unix(); task.ExecTime < now {
		task.ExecTime = now
	}

	if task.Id == "" {
		task.Id = util.NewId()

	} else if !util.IsIdValid(task.Id) {
		return contracts.TmErrorUuidIsNotCorrect
	}

	err := s.retry(func() error {
		return s.repository.Create(*task, isTaken)
	}, contracts.Deadlock)

	if err == contracts.TaskExist {
		s.eeh.New(contracts.LevelDebug, err.Error(), map[string]interface{}{
			"task": task,
		})

		return contracts.TmErrorTaskExist
	} else if err != nil {
		s.eeh.New(contracts.LevelError, err.Error(), map[string]interface{}{
			"task": task,
		})

		return contracts.TmErrorCreatingTasks
	}

	return nil
}

func (s *taskManager) Delete(taskId string) (err error) {

	errDeleting := s.retry(func() error {
		return s.repository.Delete([]domain.Task{{Id: taskId}})
	}, contracts.Deadlock)

	if errDeleting != nil {
		s.eeh.New(contracts.LevelError, errDeleting.Error(), map[string]interface{}{
			"taskId": taskId,
		})
		err = contracts.TmErrorDeletingTask
	}

	return
}

func (s *taskManager) GetTasksToComplete(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error) {
	var collections contracts.CollectionsInterface
	errFinding := s.retry(func() (err error) {
		collections, err = s.repository.FindBySecToExecTime(preloadingTimeRange)
		return
	}, contracts.Deadlock)

	switch {
	case errFinding == contracts.NoTasksFound:
		return nil, contracts.TmErrorCollectionsNotFound
	case errFinding != nil:
		s.eeh.New(contracts.LevelError, errFinding.Error(), nil)
		return collections, contracts.TmErrorGetTasks
	}

	return collections, nil
}

func (s *taskManager) ConfirmExecution(task []domain.Task) error {

	errConfirm := s.retry(func() error {
		return s.repository.Delete(task)
	}, contracts.Deadlock)

	if errConfirm != nil {
		s.eeh.New(contracts.LevelError, errConfirm.Error(), map[string]interface{}{
			"task": task,
		})
		return contracts.TmErrorConfirmationTasks
	}

	return nil
}

func (s *taskManager) retry(callback func() error, retryableErrors ...error) (err error) {
	for try := 1; try <= s.maxRetry; try++ {
		if err = callback(); err != nil {
			if contains(retryableErrors, err) {
				s.eeh.New(contracts.LevelError, err.Error(), map[string]interface{}{
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

func contains(s []error, e error) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
