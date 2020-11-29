package services

import (
	"errors"
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain"
	"github.com/VladislavPav/trigger-hook/utils"
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
