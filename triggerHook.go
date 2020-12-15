package triggerHook

import (
	"github.com/google/uuid"
	"github.com/pvelx/triggerHook/clients"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/repository"
	"github.com/pvelx/triggerHook/services"
)

var appInstanceId string

func init() {
	appInstanceId = uuid.New().String()
}

func Default() *triggerHook {
	chPreloadedTasks := make(chan domain.Task, 1000000)
	chTasksReadyToSend := make(chan domain.Task, 1000000)
	repo := repository.NewRepository(clients.Client, appInstanceId)
	taskManager := services.NewTaskManager(repo)
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

func (s *triggerHook) Delete(taskId int64) (bool, error) {
	if err := s.taskManager.Delete(domain.Task{Id: taskId}); err != nil {
		return false, err
	}
	return true, nil
}

func (s *triggerHook) Create(execTime int64) (*domain.Task, error) {
	return s.preloadingTaskService.AddNewTask(execTime)
}

func (s *triggerHook) Run() {
	go s.preloadingTaskService.Preload()
	go s.senderService.Send()
	s.waitingTaskService.WaitUntilExecTime()
}
