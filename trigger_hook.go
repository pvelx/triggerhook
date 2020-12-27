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
	eventErrorHandler := services.NewEventErrorHandler()

	repo := repository.NewRepository(client, appInstanceId, eventErrorHandler, nil)
	if err := repo.Up(); err != nil {
		panic(err)
	}

	taskManager := services.NewTaskManager(repo)
	preloadingTaskService := services.NewPreloadingTaskService(taskManager)

	waitingTaskService := services.NewWaitingTaskService(preloadingTaskService.GetPreloadedChan())

	senderService := services.NewTaskSender(
		taskManager,
		waitingTaskService.GetReadyToSendChan(),
		nil,
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

func (s *triggerHook) SetErrorHandler(externalErrorHandler func(event contracts.EventError)) {
	s.eventErrorHandler.SetErrorHandler(externalErrorHandler)
}

func (s *triggerHook) Delete(task domain.Task) (bool, error) {
	if err := s.taskManager.Delete(task); err != nil {
		return false, err
	}
	s.waitingTaskService.CancelIfExist(task.Id)

	return true, nil
}

func (s *triggerHook) Create(task domain.Task) error {
	return s.preloadingTaskService.AddNewTask(task)
}

func (s *triggerHook) Run() error {
	go s.preloadingTaskService.Preload()
	go s.senderService.Send()
	go s.waitingTaskService.WaitUntilExecTime()

	return s.eventErrorHandler.Listen()
}
