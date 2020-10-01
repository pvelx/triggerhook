package tasks

import (
	"errors"
	"github.com/VladislavPav/trigger-hook/repository"
)

type Service interface {
	Create(Task) (*Task, *error)
	Delete(Task) (*Task, *error)
	UpdateTask(Task) (*Task, *error)
	FindToExec(secToLaunch int64) ([]*Task, *error)
	UpdateNextExecTime(Task) ([]*Task, *error)
}

func NewService(repo repository.RepositoryInterface) Service {
	return &service{
		repo: repo,
	}
}

type service struct {
	repo repository.RepositoryInterface
}

func (s *service) Create(task Task) (*Task, *error) {
	if err := s.repo.Create(task); err != nil {
		err := errors.New(err.Error())
		return nil, &err
	}
	return &task, nil
}

func (s *service) Delete(task Task) (*Task, *error) {

	return nil, nil
}

func (s *service) FindToExec(secToLaunch int64) ([]*Task, *error) {

	return nil, nil
}

func (s *service) UpdateTask(task Task) (*Task, *error) {

	return nil, nil
}

func (s *service) UpdateNextExecTime(task Task) ([]*Task, *error) {

	return nil, nil
}
