package triggerHook

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
)

func New(
	eventErrorHandler contracts.EventErrorHandlerInterface,
	waitingTaskService contracts.WaitingTaskServiceInterface,
	preloadingTaskService contracts.PreloadingTaskServiceInterface,
	senderService contracts.TaskSenderInterface,
	monitoringService contracts.MonitoringInterface,
) contracts.TasksDeferredInterface {

	return &triggerHook{
		eventErrorHandler:     eventErrorHandler,
		waitingTaskService:    waitingTaskService,
		preloadingTaskService: preloadingTaskService,
		senderService:         senderService,
		monitoringService:     monitoringService,
	}
}

type triggerHook struct {
	waitingTaskService    contracts.WaitingTaskServiceInterface
	preloadingTaskService contracts.PreloadingTaskServiceInterface
	senderService         contracts.TaskSenderInterface
	eventErrorHandler     contracts.EventErrorHandlerInterface
	monitoringService     contracts.MonitoringInterface
}

func (s *triggerHook) Delete(taskId string) error {
	return s.waitingTaskService.CancelIfExist(taskId)
}

func (s *triggerHook) Create(task *domain.Task) error {
	return s.preloadingTaskService.AddNewTask(task)
}

func (s *triggerHook) Consume() contracts.TaskToSendInterface {
	return s.senderService.Consume()
}

func (s *triggerHook) Run() error {
	go s.preloadingTaskService.Run()
	go s.waitingTaskService.Run()
	go s.senderService.Run()
	go s.monitoringService.Run()

	return s.eventErrorHandler.Run()
}
