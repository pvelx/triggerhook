package repository

import (
	"context"
	"time"

	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

type RepositoryMock struct {
	contracts.RepositoryInterface

	/*
		You need to substitute *Mock methods to do substitute original functions
	*/
	CreateMock              func(ctx context.Context, task domain.Task, isTaken bool) error
	DeleteMock              func(ctx context.Context, tasks []domain.Task) (int64, error)
	FindBySecToExecTimeMock func(ctx context.Context, preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error)
	UpMock                  func() error
	CountMock               func() (int, error)
}

func (r *RepositoryMock) Create(ctx context.Context, task domain.Task, isTaken bool) error {
	return r.CreateMock(ctx, task, isTaken)
}

func (r *RepositoryMock) Delete(ctx context.Context, tasks []domain.Task) (int64, error) {
	return r.DeleteMock(ctx, tasks)
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

func (r *RepositoryMock) FindBySecToExecTime(ctx context.Context, preloadingTimeRange time.Duration) (
	collection contracts.CollectionsInterface,
	error error,
) {
	return r.FindBySecToExecTimeMock(ctx, preloadingTimeRange)
}

type CollectionsMock struct {
	contracts.CollectionsInterface
	NextMock func(ctx context.Context) (tasks []domain.Task, err error)
}

func (c *CollectionsMock) Next(ctx context.Context) (tasks []domain.Task, err error) {
	return c.NextMock(ctx)
}
