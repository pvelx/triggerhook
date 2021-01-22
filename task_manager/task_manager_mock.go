package task_manager

import (
	"time"

	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

type TaskManagerMock struct {
	contracts.TaskManagerInterface

	/*
		You need to substitute *Mock methods to do substitute original functions
	*/
	ConfirmExecutionMock   func(tasks []domain.Task) error
	CreateMock             func(task *domain.Task, isTaken bool) error
	GetTasksToCompleteMock func(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error)
}

func (tm *TaskManagerMock) ConfirmExecution(tasks []domain.Task) error {
	return tm.ConfirmExecutionMock(tasks)
}

func (tm *TaskManagerMock) Create(task *domain.Task, isTaken bool) error {
	return tm.CreateMock(task, isTaken)
}

func (tm *TaskManagerMock) GetTasksToComplete(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error) {
	return tm.GetTasksToCompleteMock(preloadingTimeRange)
}
