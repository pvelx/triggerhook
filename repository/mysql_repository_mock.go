package repository

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

type RepositoryMock struct {
	contracts.RepositoryInterface

	/*
		You need to substitute *Mock methods to do substitute original functions
	*/
	CreateMock              func(task domain.Task, isTaken bool) error
	DeleteMock              func(tasks []domain.Task) error
	FindBySecToExecTimeMock func(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error)
	UpMock                  func() error
}

func (r *RepositoryMock) Create(task domain.Task, isTaken bool) error {
	return r.CreateMock(task, isTaken)
}

func (r *RepositoryMock) Delete(tasks []domain.Task) (error error) {
	return r.DeleteMock(tasks)
}

func (r *RepositoryMock) Up() (error error) {
	return r.UpMock()
}

func (r *RepositoryMock) FindBySecToExecTime(preloadingTimeRange time.Duration) (
	collection contracts.CollectionsInterface,
	error error,
) {
	return r.FindBySecToExecTimeMock(preloadingTimeRange)
}
