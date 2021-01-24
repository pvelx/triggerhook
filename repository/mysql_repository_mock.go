package repository

import (
	"time"

	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

type RepositoryMock struct {
	contracts.RepositoryInterface

	/*
		You need to substitute *Mock methods to do substitute original functions
	*/
	CreateMock              func(task domain.Task, isTaken bool) error
	DeleteMock              func(tasks []domain.Task) (int64, error)
	FindBySecToExecTimeMock func(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error)
	UpMock                  func() error
	CountMock               func() (int, error)
}

func (r *RepositoryMock) Create(task domain.Task, isTaken bool) error {
	return r.CreateMock(task, isTaken)
}

func (r *RepositoryMock) Delete(tasks []domain.Task) (int64, error) {
	return r.DeleteMock(tasks)
}

func (r *RepositoryMock) Up() (error error) {
	if r.UpMock == nil {
		return nil
	}
	return r.UpMock()
}

func (r *RepositoryMock) Count() (int, error) {
	if r.UpMock == nil {
		return 0, nil
	}
	return r.CountMock()
}

func (r *RepositoryMock) FindBySecToExecTime(preloadingTimeRange time.Duration) (
	collection contracts.CollectionsInterface,
	error error,
) {
	return r.FindBySecToExecTimeMock(preloadingTimeRange)
}

type CollectionsMock struct {
	contracts.CollectionsInterface
	NextMock func() (tasks []domain.Task, err error)
}

func (c *CollectionsMock) Next() (tasks []domain.Task, err error) {
	return c.NextMock()
}
