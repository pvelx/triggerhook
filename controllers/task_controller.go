package controllers

import (
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"github.com/VladislavPav/trigger-hook/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

func NewHandler(service services.TaskHandlerServiceInterface) HandlerInterface {
	return &handler{service: service}
}

type HandlerInterface interface {
	Create(c *gin.Context)
	Delete(c *gin.Context)
}

type handler struct {
	service services.TaskHandlerServiceInterface
}

func (h *handler) Create(c *gin.Context) {
	var taskRequest taskRequest
	if err := c.ShouldBindJSON(&taskRequest); err != nil {
		c.JSON(http.StatusInternalServerError, "Server error")
		return
	}
	if err := taskRequest.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, "Validation error")
		return
	}

	task := tasks.Task{
		ExecTime: taskRequest.NextExecTime,
	}

	h.service.Create(task)

	c.JSON(http.StatusOK, task)
}

func (h *handler) Delete(c *gin.Context) {
	panic("implement me")
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
