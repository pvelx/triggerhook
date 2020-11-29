package contracts

import (
	"github.com/VladislavPav/trigger-hook/domain"
	"github.com/VladislavPav/trigger-hook/utils"
)

type PrioritizedTaskListInterface interface {
	Add(task *domain.Task)
	Take() *domain.Task
}

type TaskSenderInterface interface {
	Send()
}

type SendingTransportInterface interface {
	Send(task *domain.Task) bool
}

type TaskManagerInterface interface {
	Create(task *domain.Task, isTaken bool) *error
	GetTasksBySecToExecTime(secToExecTime int64) []domain.Task
	ConfirmExecution(task *domain.Task) *utils.ErrorRepo
}

type RepositoryInterface interface {
	Create(task *domain.Task, isTaken bool) *utils.ErrorRepo
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
	Run()
}
