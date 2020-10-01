package services

import (
	"github.com/VladislavPav/trigger-hook/domain/tasks"
)

type TaskHandlerServiceInterface interface {
	Create(tasks.Task)
	Delete(tasks.Task)
	Execute()
}

func NewTaskHandlerService(taskService tasks.Service) TaskHandlerServiceInterface {
	return &taskHandlerService{taskService: taskService}
}

type taskHandlerService struct {
	taskService tasks.Service
}

func (s *taskHandlerService) Create(task tasks.Task) {
	s.taskService.Create(task)
	panic("implement me")
}

func (s *taskHandlerService) Delete(tasks.Task) {

	panic("implement me")
}

func (s *taskHandlerService) Execute() {

	panic("implement me")
}
