package services

import (
	"errors"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/utils"
)

func NewTaskManager(repo contracts.RepositoryInterface) contracts.TaskManagerInterface {
	return &taskManager{repo: repo}
}

type taskManager struct {
	contracts.TaskManagerInterface
	repo contracts.RepositoryInterface
}

func (s *taskManager) Create(task *domain.Task, isTaken bool) *error {
	if err := s.repo.Create(task, isTaken); err != nil {
		err := errors.New(err.Error())
		return &err
	}
	return nil
}

func (s *taskManager) Delete(taskId string) *error {
	if err := s.repo.Delete(taskId); err != nil {
		return err
	}
	return nil
}

func (s *taskManager) GetTasksBySecToExecTime(secToExecTime int64) []domain.Task {
	tasksToExec, err := s.repo.FindBySecToExecTime(secToExecTime)
	if err != nil {
		panic(err)
	}
	return tasksToExec
}

func (s *taskManager) ConfirmExecution(task *domain.Task) *utils.ErrorRepo {
	return s.repo.ChangeStatusToCompleted(task)
}
