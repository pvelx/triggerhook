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
		Count of tasks in the list
	*/
	Len() int

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

/*	--------------------------------------------------
	Task sender
*/

type TaskSenderInterface interface {
	Run()
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
	TmErrorTaskNotFound        = errors.New("task not found")
)

/*	--------------------------------------------------
	Repository
*/
type RepositoryInterface interface {
	Create(task domain.Task, isTaken bool) error
	Delete(tasks []domain.Task) (int64, error)
	FindBySecToExecTime(preloadingTimeRange time.Duration) (CollectionsInterface, error)
	Up() error
	Count() (int, error)
}

type CollectionsInterface interface {
	Next() (tasks []domain.Task, err error)
}

var (
	FailCountingTasks = errors.New("counting the task was fail")
	FailCreatingTask  = errors.New("creating the task was fail")
	FailDeletingTask  = errors.New("deleting the task was fail")
	FailGettingTasks  = errors.New("getting the tasks were fail")
	FailFindingTasks  = errors.New("finding the tasks were fail")
	NoTasksFound      = errors.New("no tasks found")
	NoCollections     = errors.New("collections are over")
	TaskExist         = errors.New("task with the uuid already exist")
	Deadlock          = errors.New("deadlock, please retry")
	FailSchemaSetup   = errors.New("schema setup failed")
)

/*	--------------------------------------------------
	Preloading task service
*/
type PreloadingTaskServiceInterface interface {
	AddNewTask(task *domain.Task) error
	GetPreloadedChan() <-chan domain.Task
	Run()
}

/*	--------------------------------------------------
	Waiting task service
*/
type WaitingTaskServiceInterface interface {
	CancelIfExist(taskId string) error
	GetReadyToSendChan() <-chan domain.Task
	Run()
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
		Throws new event. May be used parallel
	*/
	New(level Level, eventMessage string, extra map[string]interface{})

	/*
		Launch the event error handler
	*/
	Run() error
}

/*	--------------------------------------------------
	Monitoring
*/

type MetricType int

const (
	/*
		Measures the value at regular intervals
	*/
	ValueMetricType MetricType = iota

	/*
		Measures the number of elements in a time period
	*/
	VelocityMetricType

	/*
		Measures the number of elements in a time period
	*/
	IntegralMetricType
)

var (
	TopicIsNotInitialized    = errors.New("the topic is not initialized")
	TopicExist               = errors.New("the topic exist")
	IncorrectMeasurementType = errors.New("incorrect measurement type")
)

type MeasurementEvent struct {
	Measurement   int64
	Time          time.Time
	PeriodMeasure time.Duration
}

type Topic string

type MonitoringInterface interface {
	/*
		Init measurement
	*/
	Init(topic Topic, metricType MetricType) error

	/*
		Publishing measurement events
	*/
	Publish(topic Topic, measurement int64) error

	/*
		Listening to the measurement with periodMeasure
	*/
	Listen(topic Topic, measurement interface{}) error

	/*
		Launch monitoring
	*/
	Run()
}

/*	--------------------------------------------------
	Trigger hook interface
*/

var (
	/*
		Number of tasks waiting for confirmation after sending
	*/
	WaitingForConfirmation Topic = "waitingForConfirmation"

	/*
		The rate of confirmation of the sending task
	*/
	SpeedOfConfirmation Topic = "speedOfConfirmation"

	/*
		Number of tasks waiting to be sent
	*/
	Preloaded Topic = "preloaded"

	/*
		Speed of preloading
	*/
	SpeedOfPreloading Topic = "speedOfPreloading"

	/*
		Tasks ready to send
	*/
	CountOfWaitingForSending Topic = "countOfWaitingForSending"

	SpeedOfCreating Topic = "speedOfCreating"

	SpeedOfDeleting Topic = "speedOfDeleting"

	SpeedOfSending Topic = "speedOfSending"

	CountOfAllTasks Topic = "countOfAllTasks"
)

type TasksDeferredInterface interface {
	Create(task *domain.Task) error

	Delete(taskId string) error

	/*
		LAUNCHER TRIGGER HOOK :) !!!
	*/
	Run() error
}
