package triggerhook

import (
	"github.com/pvelx/triggerhook/connection"
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/error_service"
	"github.com/pvelx/triggerhook/monitoring_service"
	"github.com/pvelx/triggerhook/preloader_service"
	"github.com/pvelx/triggerhook/repository"
	"github.com/pvelx/triggerhook/sender_service"
	"github.com/pvelx/triggerhook/task_manager"
	"github.com/pvelx/triggerhook/util"
	"github.com/pvelx/triggerhook/waiting_service"
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

func Build(config Config) contracts.TriggerHookInterface {

	errorService := error_service.New(&config.ErrorServiceOptions)
	monitoringService := monitoring_service.New(&config.MonitoringServiceOptions)

	repositoryService := repository.New(
		connection.New(&config.Connection),
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
		&config.SenderServiceOptions,
	)

	return New(
		errorService,
		waitingService,
		preloaderService,
		senderService,
		monitoringService,
	)
}
