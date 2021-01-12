package triggerHook

import (
	"github.com/pvelx/triggerHook/connection"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/error_service"
	"github.com/pvelx/triggerHook/monitoring_service"
	"github.com/pvelx/triggerHook/preloader_service"
	"github.com/pvelx/triggerHook/repository"
	"github.com/pvelx/triggerHook/sender_service"
	"github.com/pvelx/triggerHook/task_manager"
	"github.com/pvelx/triggerHook/util"
	"github.com/pvelx/triggerHook/waiting_service"
)

type Config struct {
	Connection               connection.Options
	RepositoryOptions        repository.Options
	ErrorServiceOptions      error_service.Options
	MonitoringServiceOptions monitoring_service.Options
	SenderServiceOptions     sender_service.Options
	WaitingServiceOptions    waiting_service.Options
	TaskManagerOptions       task_manager.Options
	PreloaderServiceOptions  preloader_service.Options
}

func Build(config Config) contracts.TasksDeferredInterface {

	errorService := error_service.New(&config.ErrorServiceOptions)
	monitoringService := monitoring_service.New(&config.MonitoringServiceOptions)

	repositoryService := repository.New(
		connection.NewMysqlClient(config.Connection),
		util.NewId(),
		errorService,
		&config.RepositoryOptions,
	)

	taskManager := task_manager.New(
		repositoryService,
		errorService,
		monitoringService,
		&config.TaskManagerOptions,
	)

	preloaderService := preloader_service.New(
		taskManager,
		errorService,
		monitoringService,
		&config.PreloaderServiceOptions,
	)

	waitingService := waiting_service.New(
		preloaderService.GetPreloadedChan(),
		monitoringService,
		taskManager,
		errorService,
		&config.WaitingServiceOptions,
	)

	senderService := sender_service.New(
		taskManager,
		waitingService.GetReadyToSendChan(),
		errorService,
		monitoringService,
		config.SenderServiceOptions,
	)

	return New(
		errorService,
		waitingService,
		preloaderService,
		senderService,
		monitoringService,
	)
}
