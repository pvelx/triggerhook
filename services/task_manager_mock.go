package services

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

type taskManagerMock struct {
	contracts.TaskManagerInterface
	confirmExecutionMock   func(tasks []domain.Task) error
	createMock             func(task domain.Task, isTaken bool) error
	getTasksToCompleteMock func(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error)
}

func (tm *taskManagerMock) ConfirmExecution(tasks []domain.Task) error {
	return tm.confirmExecutionMock(tasks)
}

func (tm *taskManagerMock) Create(task domain.Task, isTaken bool) error {
	return tm.createMock(task, isTaken)
}

func (tm *taskManagerMock) GetTasksToComplete(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error) {
	return tm.getTasksToCompleteMock(preloadingTimeRange)
}

type collectionsMock struct {
	contracts.CollectionsInterface
	nextMock func() (tasks []domain.Task, isEnd bool, err error)
}

func (c *collectionsMock) Next() (tasks []domain.Task, isEnd bool, err error) {
	return c.nextMock()
}
