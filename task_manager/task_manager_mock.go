package task_manager

import (
	"context"
	"time"

	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

type TaskManagerMock struct {
	contracts.TaskManagerInterface

	/*
		You need to substitute *Mock methods to do substitute original functions
	*/
	ConfirmExecutionMock   func(ctx context.Context, tasks []domain.Task) error
	CreateMock             func(ctx context.Context, task *domain.Task, isTaken bool) error
	DeleteMock             func(ctx context.Context, taskId string) error
	GetTasksToCompleteMock func(ctx context.Context, preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error)
}

func (tm *TaskManagerMock) ConfirmExecution(ctx context.Context, tasks []domain.Task) error {
	return tm.ConfirmExecutionMock(ctx, tasks)
}

func (tm *TaskManagerMock) Create(ctx context.Context, task *domain.Task, isTaken bool) error {
	return tm.CreateMock(ctx, task, isTaken)
}

func (tm *TaskManagerMock) GetTasksToComplete(ctx context.Context, preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error) {
	return tm.GetTasksToCompleteMock(ctx, preloadingTimeRange)
}

func (tm *TaskManagerMock) Delete(ctx context.Context, taskId string) error {
	return tm.DeleteMock(ctx, taskId)
}
