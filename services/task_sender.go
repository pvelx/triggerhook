package services

import (
	"fmt"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
)

func NewTaskSender(
	taskManager contracts.TaskManagerInterface,
	chTasksReadyToSend <-chan domain.Task,
) contracts.TaskSenderInterface {
	return &taskSender{
		taskManager:        taskManager,
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

func (s *taskSender) SetTransport(transport contracts.SendingTransportInterface) {
	s.transport = transport
}

func (s *taskSender) Send() {
	if s.transport == nil {
		panic("Transport for sending was not added")
	}
	go s.confirm()
	for {
		select {
		case task := <-s.chTasksReadyToSend:
			if s.transport.Send(&task) {
				s.chTasksToConfirm <- task
			}
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