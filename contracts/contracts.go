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
	DeleteIfExist(taskId string) bool
}

type TaskSenderInterface interface {
	Send()

	//Setting the function with the desired message sending method
	SetTransport(func(task *domain.Task))
}

type TaskManagerInterface interface {
	Create(task []TaskToCreate) error
	Delete(task []*domain.Task) error
	GetTasksToComplete(secToExecTime int64) (CollectionsInterface, error)
	ConfirmExecution(task []*domain.Task) error
}

type TaskToCreate struct {
	Task    *domain.Task
	IsTaken bool
}

type CollectionsInterface interface {
	Next() (tasks []domain.Task, isEnd bool, err error)
}
type RepositoryInterface interface {
	Create(tasks []TaskToCreate) error
	Delete(tasks []*domain.Task) error
	ClearEmptyCollection() error
	FindBySecToExecTime(secToNow int64) (CollectionsInterface, error)
	Up() error
}

type PreloadingTaskServiceInterface interface {
	AddNewTask(tasks []*domain.Task) error
	GetPreloadedChan() <-chan domain.Task
	Preload()
}

type WaitingTaskServiceInterface interface {
	WaitUntilExecTime()
	CancelIfExist(taskId string)
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
	Create(tasks []*domain.Task) error
	Delete(tasks []*domain.Task) (bool, error)

	//Setting the function with the desired message sending method (for example, RabbitMQ)
	//YOU SHOULD TAKE CARE HANDLE EXCEPTIONS WHILE SEND IN THIS FUNCTION
	SetTransport(func(task *domain.Task))

	//Configure the function using the desired error handler (for example, file logger, Sentry or other)
	SetErrorHandler(func(EventError))

	//LAUNCHER TRIGGER HOOK :) !!!
	Run() error
}
