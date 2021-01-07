package contracts

import (
	"errors"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

/*	--------------------------------------------------
	Prioritized task list
*/

type PrioritizedTaskListInterface interface {

	/*
		Add a task to the list based on priority
	*/
	Add(task domain.Task)

	/*
		Take the most prioritized task and delete from list
	*/
	Take() *domain.Task

	/*
		Searches for a task and deletes it return true when the task was deleted, false - when was not found
	*/
	DeleteIfExist(taskId string) bool
}

type TaskSenderInterface interface {
	Send()

	/*
		Setting the function with the desired message sending method
	*/
	SetTransport(func(task domain.Task))
}

/*	--------------------------------------------------
	Task manager
*/
type TaskManagerInterface interface {
	Create(task *domain.Task, isTaken bool) error
	Delete(taskId string) error
	GetTasksToComplete(preloadingTimeRange time.Duration) (CollectionsInterface, error)
	ConfirmExecution(task []domain.Task) error
}

var (
	TmErrorCreatingTasks       = errors.New("cannot create task")
	TmErrorUuidIsNotCorrect    = errors.New("uuid of the task is not correct")
	TmErrorTaskExist           = errors.New("task with this uuid already exist")
	TmErrorConfirmationTasks   = errors.New("cannot confirm execution of tasks")
	TmErrorGetTasks            = errors.New("cannot get any tasks")
	TmErrorCollectionsNotFound = errors.New("collections not found")
	TmErrorDeletingTask        = errors.New("cannot delete task")
)

/*	--------------------------------------------------
	Repository
*/
type RepositoryInterface interface {
	Create(task domain.Task, isTaken bool) error
	Delete(tasks []domain.Task) error
	FindBySecToExecTime(preloadingTimeRange time.Duration) (CollectionsInterface, error)
	Up() error
}

type CollectionsInterface interface {
	Next() (tasks []domain.Task, err error)
}

var (
	FailCreatingTask = errors.New("creating the task was fail")
	FailDeletingTask = errors.New("deleting the task was fail")
	FailGettingTasks = errors.New("getting the tasks were fail")
	FailFindingTasks = errors.New("finding the tasks were fail")
	NoTasksFound     = errors.New("no tasks found")
	NoCollections    = errors.New("collections are over")
	TaskExist        = errors.New("task with the uuid already exist")
	Deadlock         = errors.New("deadlock, please retry")
	FailSchemaSetup  = errors.New("schema setup failed")
)

/*	--------------------------------------------------
	Preloading task service
*/
type PreloadingTaskServiceInterface interface {
	AddNewTask(task *domain.Task) error
	GetPreloadedChan() <-chan domain.Task
	Preload()
}

/*	--------------------------------------------------
	Waiting task service
*/
type WaitingTaskServiceInterface interface {
	WaitUntilExecTime()
	CancelIfExist(taskId string)
	GetReadyToSendChan() <-chan domain.Task
}

/*	--------------------------------------------------
	Event error handler
*/

type Level int

const (
	/*
		The application cannot continue
	*/
	LevelFatal Level = iota

	/*
		Must be delivered to support
	*/
	LevelError

	/*
		Must be disabled in production
	*/
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

	/*
		YOU SHOULD TAKE CARE HANDLE EVENT, FOR EXAMPLE, FOR WRITE A LOG
	*/
	SetErrorHandler(level Level, eventHandler func(event EventError))

	/*
		Throws new event. May be used parallel
	*/
	New(level Level, eventMessage string, extra map[string]interface{})

	/*
		Launch the event error handler
	*/
	Listen() error
}

/*	--------------------------------------------------
	Monitoring
*/

type MetricType int

const (
	/*
		Measures the value at regular intervals
	*/
	Value MetricType = iota

	/*
		Measures the number of elements in a time period
	*/
	Velocity
)

type SubscriptionInterface interface {
	Close()
}

type MeasurementEvent struct {
	Measurement   int64
	Time          time.Time
	PeriodMeasure time.Duration
}

type MonitoringInterface interface {
	/*
		Init measurement
	*/
	Init(topic string, metricType MetricType)

	/*
		Publishing measurement events
	*/
	Pub(topic string, measurement int64) error

	/*
		Subscribe to events of measure
	*/
	Sub(topic string, callback func(measurementEvent MeasurementEvent)) (SubscriptionInterface, error)

	/*
		Launch monitoring
	*/
	Run()
}

/*  --------------------------------------------------
Trigger hook interface
*/
type TasksDeferredInterface interface {
	Create(task *domain.Task) error
	Delete(taskId string) error

	//Setting the function with the desired message sending method (for example, RabbitMQ)
	//YOU SHOULD TAKE CARE HANDLE EXCEPTIONS WHILE SEND IN THIS FUNCTION
	SetTransport(func(task domain.Task))

	//Configure the function using the desired error handler (for example, file logger, Sentry or other)
	SetErrorHandler(Level, func(EventError))

	//LAUNCHER TRIGGER HOOK :) !!!
	Run() error
}
