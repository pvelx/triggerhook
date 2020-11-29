package main

import (
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain"
	"github.com/VladislavPav/trigger-hook/repository"
	"github.com/VladislavPav/trigger-hook/services"
	"github.com/VladislavPav/trigger-hook/transport"
)

func Default() *Scheduler {
	chPreloadedTasks := make(chan domain.Task, 1000000)
	chTasksReadyToSend := make(chan domain.Task, 1000000)
	amqpTransport := transport.NewTransportAmqp()
	taskManager := services.NewTaskManager(repository.MysqlRepo)
	preloadingTaskService := services.NewPreloadingTaskService(taskManager, chPreloadedTasks)
	waitingTaskService := services.NewWaitingTaskService(chPreloadedTasks, chTasksReadyToSend)
	senderService := services.NewTaskSender(taskManager, amqpTransport, chTasksReadyToSend)

	return &Scheduler{
		chPreloadedTasks,
		chTasksReadyToSend,
		waitingTaskService,
		preloadingTaskService,
		senderService,
	}
}

type Scheduler struct {
	chPreloadedTasks      chan domain.Task
	chTasksReadyToSend    chan domain.Task
	waitingTaskService    contracts.WaitingTaskServiceInterface
	preloadingTaskService contracts.PreloadingTaskServiceInterface
	senderService         contracts.TaskSenderInterface
}

func (s *Scheduler) Create(execTime int64) (*domain.Task, *error) {
	return s.preloadingTaskService.AddNewTask(execTime)
}

func (s *Scheduler) Run() {
	go s.preloadingTaskService.Preload()
	go s.senderService.Send()
	s.waitingTaskService.WaitUntilExecTime()
}
