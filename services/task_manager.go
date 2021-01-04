package services

import (
	"fmt"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
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

func (s *taskManager) Create(task domain.Task, isTaken bool) (err error) {
	if now := time.Now().Unix(); task.ExecTime < now {
		task.ExecTime = now
	}

	errCreating := s.retry(func() error {
		return s.repository.Create(task, isTaken)
	}, contracts.Deadlock)

	if errCreating != nil {
		s.eeh.New(contracts.LevelError, errCreating.Error(), map[string]interface{}{
			"task": task,
		})
		err = contracts.TmErrorCreatingTasks
	}

	return
}

func (s *taskManager) Delete(task domain.Task) (err error) {

	errDeleting := s.retry(func() error {
		return s.repository.Delete([]domain.Task{task})
	}, contracts.Deadlock)

	if errDeleting != nil {
		s.eeh.New(contracts.LevelError, errDeleting.Error(), map[string]interface{}{
			"task": task,
		})
		err = contracts.TmErrorDeletingTask
	}

	return
}

func (s *taskManager) GetTasksToComplete(preloadingTimeRange time.Duration) (
	collections contracts.CollectionsInterface,
	err error,
) {
	errFinding := s.retry(func() (err error) {
		collections, err = s.repository.FindBySecToExecTime(preloadingTimeRange)
		return
	}, contracts.Deadlock)

	if errFinding != nil {
		s.eeh.New(contracts.LevelError, errFinding.Error(), nil)
		err = contracts.TmErrorGetTasks
	}

	fmt.Println(collections, err)

	return
}

func (s *taskManager) ConfirmExecution(task []domain.Task) (err error) {

	errConfirm := s.retry(func() error {
		return s.repository.Delete(task)
	}, contracts.Deadlock)

	if errConfirm != nil {
		s.eeh.New(contracts.LevelError, errConfirm.Error(), map[string]interface{}{
			"task": task,
		})
		err = contracts.TmErrorConfirmationTasks
	}

	return
}

func (s *taskManager) retry(callback func() error, retryableErrors ...error) (err error) {
	for try := 1; try <= s.maxRetry; try++ {
		if err = callback(); err != nil {

			s.eeh.New(contracts.LevelError, err.Error(), map[string]interface{}{
				"try": try,
			})

			if contains(retryableErrors, err) {
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
