package app

import (
	"github.com/VladislavPav/trigger-hook/controllers"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"github.com/VladislavPav/trigger-hook/repository"
	"github.com/VladislavPav/trigger-hook/services"
	"github.com/gin-gonic/gin"
)

var router = gin.Default()

func StartApp() {
	handler := services.NewTaskHandlerService(
		tasks.NewService(
			repository.MysqlRepo))
	controller := controllers.NewHandler(handler)
	go handler.Execute()

	router.POST("/task", controller.Create)
	router.Run(":8083")
}
