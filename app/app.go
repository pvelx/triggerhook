package app

import (
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"github.com/VladislavPav/trigger-hook/repository"
	"github.com/VladislavPav/trigger-hook/services"
)

type SchedulerInterface interface {
	Create(execTime int64) (*tasks.Task, *error)
	Run()
}

func Default() *Scheduler {
	chPreloadedTasks := make(chan tasks.Task, 1000000)
	chTasksReadyToSend := make(chan tasks.Task, 1000000)

	return &Scheduler{
		chPreloadedTasks:   chPreloadedTasks,
		chTasksReadyToSend: chTasksReadyToSend,
		taskService:        tasks.NewService(repository.MysqlRepo, chPreloadedTasks),
		expectantService:   services.NewTaskHandlerService(chPreloadedTasks, chTasksReadyToSend),
		senderService:      services.NewTaskSender(chTasksReadyToSend),
	}
}

type Scheduler struct {
	SchedulerInterface
	chPreloadedTasks   chan tasks.Task
	chTasksReadyToSend chan tasks.Task
	expectantService   services.TaskHandlerServiceInterface
	taskService        tasks.Service
	senderService      contracts.TaskSenderInterface
}

func (s *Scheduler) Create(execTime int64) (*tasks.Task, *error) {
	return s.taskService.Create(execTime)
}

func (s *Scheduler) Run() {
	go s.taskService.Preload()
	go s.senderService.Send()
	s.expectantService.WaitUntilExecTime()
}
