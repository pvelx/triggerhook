package app

import (
	"github.com/VladislavPav/trigger-hook/clients/cassandra"
	"github.com/VladislavPav/trigger-hook/controllers"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"github.com/VladislavPav/trigger-hook/repository"
	"github.com/VladislavPav/trigger-hook/services"
	"github.com/gin-gonic/gin"
)

var router = gin.Default()

func StartApp() {
	session := mysql.GetSession()
	defer session.Close()
	handler := controllers.NewHandler(
		services.NewTaskHandlerService(
			tasks.NewService(
				repository.CassandraRepo)))

	router.POST("/task", handler.Create)

	router.Run(":8082")
}
