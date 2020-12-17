package contracts

import (
	"github.com/pvelx/triggerHook/domain"
)

type PrioritizedTaskListInterface interface {

	//Add a task to the list based on priority
	Add(task *domain.Task)

	//Take the most prioritized task and delete from list
	Take() *domain.Task

	//Searches for a task and deletes it
	//return true when the task was deleted, false - when was not found
	DeleteIfExist(taskId int64) bool
}

type TaskSenderInterface interface {
	Send()

	//Setting the function with the desired message sending method
	SetTransport(func(task *domain.Task))
}

type TaskManagerInterface interface {
	Create(task *domain.Task, isTaken bool) error
	Delete(task domain.Task) error
	GetTasksBySecToExecTime(secToExecTime int64, count int) ([]domain.Task, error)
	CountReadyToExec(secToExecTime int64) (int, error)
	ConfirmExecution(task *domain.Task) error
}

type RepositoryInterface interface {
	Create(task *domain.Task, isTaken bool) error
	//DeleteBunch(tasks []*domain.Task) error
	ConfirmExecution(tasks *domain.Task) error
	FindBySecToExecTime(secToNow int64, count int) (domain.Tasks, error)
	CountReadyToExec(secToNow int64) (int, error)
	Up() error
}

type PreloadingTaskServiceInterface interface {
	AddNewTask(execTime int64) (*domain.Task, error)
	GetPreloadedChan() <-chan domain.Task
	Preload()
}

type WaitingTaskServiceInterface interface {
	WaitUntilExecTime()
	CancelIfExist(taskId int64)
	GetReadyToSendChan() <-chan domain.Task
}

type Level int

const (
	//The application cannot continue
	LevelFatal Level = iota

	//Must be delivered to support
	LevelError
)

type EventError struct {
	Level Level
	Error error
}

type EventErrorHandlerInterface interface {
	SetErrorHandler(func(EventError))
	NewEventError(level Level, error error)
	Listen() error
}

type TasksDeferredInterface interface {
	Create(execTime int64) (*domain.Task, error)
	Delete(taskId int64) (bool, error)

	//Setting the function with the desired message sending method (for example, RabbitMQ)
	//YOU SHOULD TAKE CARE HANDLE EXCEPTIONS WHILE SEND IN THIS FUNCTION
	SetTransport(func(task *domain.Task))

	//Configure the function using the desired error handler (for example, file logger, Sentry or other)
	SetErrorHandler(func(EventError))

	//LAUNCHER TRIGGER HOOK :) !!!
	Run() error
}
