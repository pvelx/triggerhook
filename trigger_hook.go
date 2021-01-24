package triggerhook

import (
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

func New(
	eventHandler contracts.EventHandlerInterface,
	waitingService contracts.WaitingServiceInterface,
	preloadingService contracts.PreloadingServiceInterface,
	senderService contracts.SenderServiceInterface,
	monitoringService contracts.MonitoringInterface,
) contracts.TriggerHookInterface {

	return &triggerHook{
		eventHandler:      eventHandler,
		waitingService:    waitingService,
		preloadingService: preloadingService,
		senderService:     senderService,
		monitoringService: monitoringService,
	}
}

type triggerHook struct {
	waitingService    contracts.WaitingServiceInterface
	preloadingService contracts.PreloadingServiceInterface
	senderService     contracts.SenderServiceInterface
	eventHandler      contracts.EventHandlerInterface
	monitoringService contracts.MonitoringInterface
}

func (s *triggerHook) Delete(taskId string) error {
	return s.waitingService.CancelIfExist(taskId)
}

func (s *triggerHook) Create(task *domain.Task) error {
	return s.preloadingService.AddNewTask(task)
}

func (s *triggerHook) Consume() contracts.TaskToSendInterface {
	return s.senderService.Consume()
}

func (s *triggerHook) Run() error {
	go s.preloadingService.Run()
	go s.waitingService.Run()
	go s.senderService.Run()
	go s.monitoringService.Run()

	return s.eventHandler.Run()
}
