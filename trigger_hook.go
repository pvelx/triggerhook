package triggerHook

import (
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

func New(
	eventErrorHandler contracts.EventErrorHandlerInterface,
	waitingService contracts.WaitingServiceInterface,
	preloadingTaskService contracts.PreloadingTaskServiceInterface,
	senderService contracts.TaskSenderInterface,
	monitoringService contracts.MonitoringInterface,
) contracts.TasksDeferredInterface {

	return &triggerHook{
		eventErrorHandler:     eventErrorHandler,
		waitingService:        waitingService,
		preloadingTaskService: preloadingTaskService,
		senderService:         senderService,
		monitoringService:     monitoringService,
	}
}

type triggerHook struct {
	waitingService        contracts.WaitingServiceInterface
	preloadingTaskService contracts.PreloadingTaskServiceInterface
	senderService         contracts.TaskSenderInterface
	eventErrorHandler     contracts.EventErrorHandlerInterface
	monitoringService     contracts.MonitoringInterface
}

func (s *triggerHook) Delete(taskId string) error {
	return s.waitingService.CancelIfExist(taskId)
}

func (s *triggerHook) Create(task *domain.Task) error {
	return s.preloadingTaskService.AddNewTask(task)
}

func (s *triggerHook) Consume() contracts.TaskToSendInterface {
	return s.senderService.Consume()
}

func (s *triggerHook) Run() error {
	go s.preloadingTaskService.Run()
	go s.waitingService.Run()
	go s.senderService.Run()
	go s.monitoringService.Run()

	return s.eventErrorHandler.Run()
}
