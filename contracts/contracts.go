package contracts

import (
	"github.com/pvelx/triggerHook/domain"
)

type PrioritizedTaskListInterface interface {
	Add(task *domain.Task)
	Take() *domain.Task
}

type SendingTransportInterface interface {
	Send(task *domain.Task) bool
}

type TaskSenderInterface interface {
	Send()
	SetTransport(transport SendingTransportInterface)
}

type TaskManagerInterface interface {
	Create(task *domain.Task, isTaken bool) error
	Delete(task domain.Task) error
	GetTasksBySecToExecTime(secToExecTime int64) []domain.Task
	ConfirmExecution(task *domain.Task) error
}

type RepositoryInterface interface {
	Create(task *domain.Task, isTaken bool) error
	DeleteBunch(tasks []*domain.Task) error
	FindBySecToExecTime(secToNow int64, count int) (domain.Tasks, error)
	Up() error
}

type PreloadingTaskServiceInterface interface {
	AddNewTask(execTime int64) (*domain.Task, error)
	Preload()
}

type WaitingTaskServiceInterface interface {
	WaitUntilExecTime()
}

type TasksDeferredInterface interface {
	Create(execTime int64) (*domain.Task, error)
	Delete(taskId int64) (bool, error)
	SetTransport(transport SendingTransportInterface)
	Run()
}
