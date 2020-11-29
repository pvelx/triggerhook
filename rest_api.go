package main

import (
	"github.com/VladislavPav/trigger-hook/app"
	"github.com/gin-gonic/gin"
	"net/http"
)

var router = gin.Default()
var scheduler = app.Default()

func main() {

	go scheduler.Run()

	router.POST("/task", func(c *gin.Context) {
		var taskRequest taskRequest
		if err := c.ShouldBindJSON(&taskRequest); err != nil {
			c.JSON(http.StatusInternalServerError, "Server error")
			return
		}
		if err := taskRequest.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, "Validation error")
			return
		}

		task, e := scheduler.Create(taskRequest.NextExecTime)
		if e != nil {
			c.JSON(http.StatusInternalServerError, "Something wrong")
			return
		}

		c.JSON(http.StatusOK, task)
	})
	router.Run(":8083")
}

type taskRequest struct {
	NextExecTime int64 `json:"exec_time"`
}

func (tr *taskRequest) Validate() error {
	return nil
}

type restErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}
