package triggerHook

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/repository"
	"github.com/pvelx/triggerHook/services"
)

func Default() *triggerHook {
	chPreloadedTasks := make(chan domain.Task, 1000000)
	chTasksReadyToSend := make(chan domain.Task, 1000000)
	taskManager := services.NewTaskManager(repository.MysqlRepo)
	preloadingTaskService := services.NewPreloadingTaskService(taskManager, chPreloadedTasks)
	waitingTaskService := services.NewWaitingTaskService(chPreloadedTasks, chTasksReadyToSend)
	senderService := services.NewTaskSender(taskManager, chTasksReadyToSend)

	return &triggerHook{
		chPreloadedTasks:      chPreloadedTasks,
		chTasksReadyToSend:    chTasksReadyToSend,
		waitingTaskService:    waitingTaskService,
		preloadingTaskService: preloadingTaskService,
		senderService:         senderService,
		taskManager:           taskManager,
	}
}

type triggerHook struct {
	chPreloadedTasks      chan domain.Task
	chTasksReadyToSend    chan domain.Task
	waitingTaskService    contracts.WaitingTaskServiceInterface
	preloadingTaskService contracts.PreloadingTaskServiceInterface
	senderService         contracts.TaskSenderInterface
	taskManager           contracts.TaskManagerInterface
	contracts.TasksDeferredInterface
}

func (s *triggerHook) SetTransport(transport contracts.SendingTransportInterface) {
	s.senderService.SetTransport(transport)
}

func (s *triggerHook) Delete(taskId string) (bool, *error) {
	if err := s.taskManager.Delete(taskId); err != nil {
		return false, err
	}
	return true, nil
}

func (s *triggerHook) Create(execTime int64) (*domain.Task, *error) {
	return s.preloadingTaskService.AddNewTask(execTime)
}

func (s *triggerHook) Run() {
	go s.preloadingTaskService.Preload()
	go s.senderService.Send()
	s.waitingTaskService.WaitUntilExecTime()
}
