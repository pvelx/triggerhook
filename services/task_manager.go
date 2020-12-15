package services

import (
	"errors"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
)

func NewTaskManager(repo contracts.RepositoryInterface) contracts.TaskManagerInterface {
	return &taskManager{repo: repo}
}

type taskManager struct {
	contracts.TaskManagerInterface
	repo contracts.RepositoryInterface
}

func (s *taskManager) Create(task *domain.Task, isTaken bool) error {
	if err := s.repo.Create(task, isTaken); err != nil {
		err := errors.New(err.Error())
		return err
	}
	return nil
}

func (s *taskManager) Delete(task domain.Task) error {
	if err := s.repo.DeleteBunch([]*domain.Task{&task}); err != nil {
		return err
	}
	return nil
}

func (s *taskManager) CountReadyToExec(secToExecTime int64) (int, error) {
	return s.repo.CountReadyToExec(secToExecTime)
}

func (s *taskManager) GetTasksBySecToExecTime(secToExecTime int64, count int) ([]domain.Task, error) {
	tasksToExec, err := s.repo.FindBySecToExecTime(secToExecTime, count)
	if err != nil {
		return nil, err
	}
	return tasksToExec, nil
}

func (s *taskManager) ConfirmExecution(task *domain.Task) error {
	return s.repo.DeleteBunch([]*domain.Task{task})
}
