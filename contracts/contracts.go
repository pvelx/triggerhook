package contracts

import (
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/utils"
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
	Create(task *domain.Task, isTaken bool) *error
	Delete(taskId string) *error
	GetTasksBySecToExecTime(secToExecTime int64) []domain.Task
	ConfirmExecution(task *domain.Task) *utils.ErrorRepo
}

type RepositoryInterface interface {
	Create(task *domain.Task, isTaken bool) *utils.ErrorRepo
	Delete(taskId string) *error
	FindBySecToExecTime(secToNow int64) (domain.Tasks, *utils.ErrorRepo)
	ChangeStatusToCompleted(*domain.Task) *utils.ErrorRepo
}

type PreloadingTaskServiceInterface interface {
	AddNewTask(execTime int64) (*domain.Task, *error)
	Preload()
}

type WaitingTaskServiceInterface interface {
	WaitUntilExecTime()
}

type SchedulerInterface interface {
	Create(execTime int64) (*domain.Task, *error)
	Delete(id string) (bool, *error)
	SetTransport(transport SendingTransportInterface)
	Run()
}
