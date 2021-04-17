package triggerhook

import (
	"context"
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

// Deprecated
func (s *triggerHook) Delete(taskID string) error {
	return s.waitingService.CancelIfExist(context.Background(), taskID)
}

// Deprecated
func (s *triggerHook) Create(task *domain.Task) error {
	return s.preloadingService.AddNewTask(context.Background(), task)
}

func (s *triggerHook) DeleteCtx(ctx context.Context, taskID string) error {
	return s.waitingService.CancelIfExist(ctx, taskID)
}

func (s *triggerHook) CreateCtx(ctx context.Context, task *domain.Task) error {
	return s.preloadingService.AddNewTask(ctx, task)
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
