package triggerHook

import (
	"database/sql"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/event_error_handler_service"
	"github.com/pvelx/triggerHook/monitoring_service"
	"github.com/pvelx/triggerHook/preloader_service"
	"github.com/pvelx/triggerHook/repository"
	"github.com/pvelx/triggerHook/task_manager"
	"github.com/pvelx/triggerHook/task_sender_service"
	"github.com/pvelx/triggerHook/util"
	"github.com/pvelx/triggerHook/waiting_service"
	"time"
)

var appInstanceId string

func init() {
	appInstanceId = util.NewId()
}

func Default(client *sql.DB) contracts.TasksDeferredInterface {
	eventErrorHandler := event_error_handler_service.New(false)

	monitoringService := monitoring_service.New(5 * time.Second)

	repo := repository.New(client, appInstanceId, eventErrorHandler, nil)
	if err := repo.Up(); err != nil {
		panic(err)
	}

	taskManager := task_manager.New(repo, eventErrorHandler)
	preloadingTaskService := preloader_service.New(taskManager, eventErrorHandler, monitoringService)

	waitingTaskService := waiting_service.New(preloadingTaskService.GetPreloadedChan(), monitoringService)

	senderService := task_sender_service.New(
		taskManager,
		waitingTaskService.GetReadyToSendChan(),
		nil,
		eventErrorHandler,
		monitoringService,
	)

	return &triggerHook{
		eventErrorHandler:     eventErrorHandler,
		waitingTaskService:    waitingTaskService,
		preloadingTaskService: preloadingTaskService,
		senderService:         senderService,
		taskManager:           taskManager,
		monitoringService:     monitoringService,
	}
}

type triggerHook struct {
	waitingTaskService    contracts.WaitingTaskServiceInterface
	preloadingTaskService contracts.PreloadingTaskServiceInterface
	senderService         contracts.TaskSenderInterface
	taskManager           contracts.TaskManagerInterface
	eventErrorHandler     contracts.EventErrorHandlerInterface
	monitoringService     contracts.MonitoringInterface
}

func (s *triggerHook) SetTransport(externalSender func(task domain.Task)) {
	s.senderService.SetTransport(externalSender)
}

func (s *triggerHook) SetErrorHandler(level contracts.Level, externalErrorHandler func(event contracts.EventError)) {
	s.eventErrorHandler.SetErrorHandler(level, externalErrorHandler)
}

func (s *triggerHook) Delete(taskId string) error {
	if err := s.taskManager.Delete(taskId); err != nil {
		return err
	}
	s.waitingTaskService.CancelIfExist(taskId)

	return nil
}

func (s *triggerHook) Create(task *domain.Task) error {
	return s.preloadingTaskService.AddNewTask(task)
}

func (s *triggerHook) Run() error {
	go s.preloadingTaskService.Preload()
	go s.senderService.Send()
	go s.waitingTaskService.WaitUntilExecTime()
	go s.monitoringService.Run()

	return s.eventErrorHandler.Listen()
}
