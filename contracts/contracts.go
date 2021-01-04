package contracts

import (
	"errors"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

type PrioritizedTaskListInterface interface {

	//Add a task to the list based on priority
	Add(task domain.Task)

	//Take the most prioritized task and delete from list
	Take() *domain.Task

	//Searches for a task and deletes it
	//return true when the task was deleted, false - when was not found
	DeleteIfExist(taskId string) bool
}

type TaskSenderInterface interface {
	Send()

	//Setting the function with the desired message sending method
	SetTransport(func(task domain.Task))
}

type TaskManagerInterface interface {
	Create(task domain.Task, isTaken bool) error
	Delete(task domain.Task) error
	GetTasksToComplete(preloadingTimeRange time.Duration) (CollectionsInterface, error)
	ConfirmExecution(task []domain.Task) error
}

var (
	TmErrorCreatingTasks     = errors.New("cannot confirm execution of tasks")
	TmErrorConfirmationTasks = errors.New("cannot confirm execution of tasks")
	TmErrorGetTasks          = errors.New("cannot get any tasks")
	TmErrorDeletingTask      = errors.New("cannot delete task")
)

type CollectionsInterface interface {
	Next() (tasks []domain.Task, isEnd bool, err error)
}

type RepositoryInterface interface {
	Create(task domain.Task, isTaken bool) error
	Delete(tasks []domain.Task) error
	FindBySecToExecTime(preloadingTimeRange time.Duration) (CollectionsInterface, error)
	Up() error
}

/*
	Repository errors
*/
var (
	FailCreatingTask = errors.New("creating the task was fail")
	FailDeletingTask = errors.New("deleting the task was fail")
	FailGettingTasks = errors.New("getting the tasks were fail")
	FailFindingTasks = errors.New("finding the tasks were fail")
	NoTasksFound     = errors.New("no tasks found")
	TaskExist        = errors.New("task with the uuid already exist")
	Deadlock         = errors.New("deadlock, please retry")
	FailSchemaSetup  = errors.New("schema setup failed")
)

type PreloadingTaskServiceInterface interface {
	AddNewTask(task domain.Task) error
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

	//Must be disabled in production
	LevelDebug
)

type EventError struct {
	Time         time.Time
	Level        Level
	EventMessage string
	Method       string
	Line         int
	File         string
	Extra        interface{}
}

type EventErrorHandlerInterface interface {

	//YOU SHOULD TAKE CARE HANDLE EVENT, FOR EXAMPLE, FOR WRITE A LOG
	SetErrorHandler(level Level, eventHandler func(event EventError))

	//Throws new event. May be used parallel
	New(level Level, eventMessage string, extra map[string]interface{})

	Listen() error
}

type TasksDeferredInterface interface {
	Create(task domain.Task) error
	Delete(task domain.Task) (bool, error)

	//Setting the function with the desired message sending method (for example, RabbitMQ)
	//YOU SHOULD TAKE CARE HANDLE EXCEPTIONS WHILE SEND IN THIS FUNCTION
	SetTransport(func(task domain.Task))

	//Configure the function using the desired error handler (for example, file logger, Sentry or other)
	SetErrorHandler(func(EventError))

	//LAUNCHER TRIGGER HOOK :) !!!
	Run() error
}
