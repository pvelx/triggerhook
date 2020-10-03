package tasks

import (
	"errors"
	"github.com/VladislavPav/trigger-hook/utils"
)

type RepositoryInterface interface {
	Create(task *Task) *utils.ErrorRepo
	Delete(task Task) *utils.ErrorRepo
	FindBySecToExecTime(secToNow int64) (Tasks, *utils.ErrorRepo)
	ChangeStatusToCompleted(*Task) *utils.ErrorRepo
}

type Service interface {
	Create(Task) (*Task, *error)
	Delete(Task) (*Task, *error)
	UpdateTask(Task) (*Task, *error)
	FindToExec(secToLaunch int64) (Tasks, error)
	UpdateNextExecTime(Task) ([]*Task, *error)
}

func NewService(repo RepositoryInterface) Service {
	return &service{
		repo: repo,
	}
}

type service struct {
	repo RepositoryInterface
}

func (s *service) Create(task Task) (*Task, *error) {
	if err := s.repo.Create(&task); err != nil {
		err := errors.New(err.Error())
		return nil, &err
	}
	return &task, nil
}

func (s *service) Delete(task Task) (*Task, *error) {

	return nil, nil
}

func (s *service) FindToExec(secToLaunch int64) (Tasks, error) {

	tasks, err := s.repo.FindBySecToExecTime(secToLaunch)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	return tasks, nil
}

func (s *service) UpdateTask(task Task) (*Task, *error) {

	return nil, nil
}

func (s *service) UpdateNextExecTime(task Task) ([]*Task, *error) {

	return nil, nil
}
