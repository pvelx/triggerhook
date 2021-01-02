package services

import (
	"errors"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

func NewTaskManager(repo contracts.RepositoryInterface) contracts.TaskManagerInterface {
	return &taskManager{repo: repo}
}

type taskManager struct {
	contracts.TaskManagerInterface
	repo contracts.RepositoryInterface
}

func (s *taskManager) Create(task domain.Task, isTaken bool) error {
	if now := time.Now().Unix(); task.ExecTime < now {
		task.ExecTime = now
	}
	if err := s.repo.Create(task, isTaken); err != nil {
		err := errors.New(err.Error())
		return err
	}
	return nil
}

func (s *taskManager) Delete(task domain.Task) error {
	if err := s.repo.Delete([]domain.Task{task}); err != nil {
		return err
	}
	return nil
}

func (s *taskManager) GetTasksToComplete(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error) {
	tasksToExec, err := s.repo.FindBySecToExecTime(preloadingTimeRange)
	if err != nil {
		return nil, err
	}
	return tasksToExec, nil
}

func (s *taskManager) ConfirmExecution(task []domain.Task) error {
	errDelete := s.repo.Delete(task)
	if errDelete != nil {
		return errDelete
	}
	return nil
}
