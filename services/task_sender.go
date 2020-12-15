package services

import (
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
	sendByExternalTransport func(task *domain.Task)
	chTasksReadyToSend      <-chan domain.Task
	chTasksToConfirm        chan domain.Task
	taskManager             contracts.TaskManagerInterface
}

func (s *taskSender) SetTransport(sendByExternalTransport func(task *domain.Task)) {
	s.sendByExternalTransport = sendByExternalTransport
}

func (s *taskSender) Send() {
	if s.sendByExternalTransport == nil {
		panic("Transport for sending was not added")
	}
	go s.confirm()
	for {
		select {
		case task := <-s.chTasksReadyToSend:
			s.sendByExternalTransport(&task)
			s.chTasksToConfirm <- task
		}
	}
}

func (s *taskSender) confirm() {
	for {
		select {
		case task, ok := <-s.chTasksToConfirm:
			if ok {
				//if task.Id%1e+6 == 0 {
				//	fmt.Println("confirmed:", task)
				//}

				if err := s.taskManager.ConfirmExecution(&task); err != nil {
					panic(err)
				}
			} else {
				panic("chan was closed")
			}
		}
	}
}
