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
	createMock              func(task domain.Task, isTaken bool) error
	deleteMock              func(tasks []domain.Task) error
	findBySecToExecTimeMock func(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error)
	upMock                  func() error
}

func (r *RepositoryMock) Create(task domain.Task, isTaken bool) error {
	return r.createMock(task, isTaken)
}

func (r *RepositoryMock) Delete(tasks []domain.Task) (error error) {
	return r.deleteMock(tasks)
}

func (r *RepositoryMock) Up() (error error) {
	return r.upMock()
}

func (r *RepositoryMock) FindBySecToExecTime(preloadingTimeRange time.Duration) (
	collection contracts.CollectionsInterface,
	error error,
) {
	return r.findBySecToExecTimeMock(preloadingTimeRange)
}
