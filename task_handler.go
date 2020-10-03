package main

import (
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"github.com/VladislavPav/trigger-hook/repository"
	"github.com/VladislavPav/trigger-hook/services"
)

func main() {
	handler := services.NewTaskHandlerService(
		tasks.NewService(
			repository.MysqlRepo))

	handler.Execute()
}
