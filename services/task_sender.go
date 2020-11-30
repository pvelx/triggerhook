package services

import (
	"fmt"
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain"
)

func NewTaskSender(
	taskManager contracts.TaskManagerInterface,
	transport contracts.SendingTransportInterface,
	chTasksReadyToSend <-chan domain.Task,
) contracts.TaskSenderInterface {
	return &taskSender{
		taskManager:        taskManager,
		transport:          transport,
		chTasksToConfirm:   make(chan domain.Task, 10000),
		chTasksReadyToSend: chTasksReadyToSend,
	}
}

type taskSender struct {
	contracts.TaskSenderInterface
	transport          contracts.SendingTransportInterface
	chTasksReadyToSend <-chan domain.Task
	chTasksToConfirm   chan domain.Task
	taskManager        contracts.TaskManagerInterface
}

func (s *taskSender) Send() {
	go s.confirm()
	for task := range s.chTasksReadyToSend {
		if s.transport.Send(&task) {
			s.chTasksToConfirm <- task
		}
	}
}

func (s *taskSender) confirm() {
	for task := range s.chTasksToConfirm {
		if task.Id%1e+6 == 0 {
			fmt.Println("confirmed:", task)
		}

		//if err := s.taskManager.ConfirmExecution(&task); err != nil {
		//	panic(err)
		//}
	}
}
