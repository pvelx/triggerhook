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

func (s *taskManager) Create(tasks []contracts.TaskToCreate) error {
	if err := s.repo.Create(tasks); err != nil {
		err := errors.New(err.Error())
		return err
	}
	return nil
}

func (s *taskManager) Delete(task domain.Task) error {
	//if err := s.repo.DeleteBunch([]*domain.Task{&task}); err != nil {
	//	return err
	//}
	return nil
}

func (s *taskManager) GetTasksBySecToExecTime(secToExecTime int64) (contracts.CollectionsInterface, error) {
	tasksToExec, err := s.repo.FindBySecToExecTime(secToExecTime)
	if err != nil {
		return nil, err
	}
	return tasksToExec, nil
}

func (s *taskManager) ConfirmExecution(task []*domain.Task) error {
	errDelete := s.repo.ConfirmExecution(task)
	if errDelete != nil {
		return errDelete
	}
	errClear := s.repo.ClearEmptyCollection()
	if errClear != nil {
		return errClear
	}
	return nil
}
