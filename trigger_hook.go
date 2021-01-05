package triggerHook

import (
	"database/sql"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/repository"
	"github.com/pvelx/triggerHook/services"
	uuid "github.com/satori/go.uuid"
)

var appInstanceId string

func init() {
	appInstanceId = uuid.NewV4().String()
}

func Default(client *sql.DB) contracts.TasksDeferredInterface {
	eventErrorHandler := services.NewEventErrorHandler(false)

	repo := repository.NewRepository(client, appInstanceId, eventErrorHandler, nil)
	if err := repo.Up(); err != nil {
		panic(err)
	}

	taskManager := services.NewTaskManager(repo, eventErrorHandler)
	preloadingTaskService := services.NewPreloadingTaskService(taskManager, eventErrorHandler)

	waitingTaskService := services.NewWaitingTaskService(preloadingTaskService.GetPreloadedChan())

	senderService := services.NewTaskSender(
		taskManager,
		waitingTaskService.GetReadyToSendChan(),
		nil,
		eventErrorHandler,
	)

	return &triggerHook{
		eventErrorHandler:     eventErrorHandler,
		waitingTaskService:    waitingTaskService,
		preloadingTaskService: preloadingTaskService,
		senderService:         senderService,
		taskManager:           taskManager,
	}
}

type triggerHook struct {
	waitingTaskService    contracts.WaitingTaskServiceInterface
	preloadingTaskService contracts.PreloadingTaskServiceInterface
	senderService         contracts.TaskSenderInterface
	taskManager           contracts.TaskManagerInterface
	eventErrorHandler     contracts.EventErrorHandlerInterface
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

	return s.eventErrorHandler.Listen()
}
