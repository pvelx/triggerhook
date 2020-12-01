package main

import (
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain"
	"github.com/VladislavPav/trigger-hook/repository"
	"github.com/VladislavPav/trigger-hook/services"
)

func Default() *Scheduler {
	chPreloadedTasks := make(chan domain.Task, 1000000)
	chTasksReadyToSend := make(chan domain.Task, 1000000)
	taskManager := services.NewTaskManager(repository.MysqlRepo)
	preloadingTaskService := services.NewPreloadingTaskService(taskManager, chPreloadedTasks)
	waitingTaskService := services.NewWaitingTaskService(chPreloadedTasks, chTasksReadyToSend)
	senderService := services.NewTaskSender(taskManager, chTasksReadyToSend)

	return &Scheduler{
		chPreloadedTasks,
		chTasksReadyToSend,
		waitingTaskService,
		preloadingTaskService,
		senderService,
		taskManager,
	}
}

type Scheduler struct {
	chPreloadedTasks      chan domain.Task
	chTasksReadyToSend    chan domain.Task
	waitingTaskService    contracts.WaitingTaskServiceInterface
	preloadingTaskService contracts.PreloadingTaskServiceInterface
	senderService         contracts.TaskSenderInterface
	taskManager           contracts.TaskManagerInterface
}

func (s *Scheduler) SetTransport(transport contracts.SendingTransportInterface) {
	s.senderService.SetTransport(transport)
}

func (s *Scheduler) Delete(taskId string) (bool, *error) {
	if err := s.taskManager.Delete(taskId); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Scheduler) Create(execTime int64) (*domain.Task, *error) {
	return s.preloadingTaskService.AddNewTask(execTime)
}

func (s *Scheduler) Run() {
	go s.preloadingTaskService.Preload()
	go s.senderService.Send()
	s.waitingTaskService.WaitUntilExecTime()
}
