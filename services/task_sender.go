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
		chTasksToConfirm:   make(chan *domain.Task, 10000000),
		chTasksReadyToSend: chTasksReadyToSend,
	}
}

type taskSender struct {
	contracts.TaskSenderInterface
	sendByExternalTransport func(task *domain.Task)
	chTasksReadyToSend      <-chan domain.Task
	chTasksToConfirm        chan *domain.Task
	taskManager             contracts.TaskManagerInterface
}

func (s *taskSender) SetTransport(sendByExternalTransport func(task *domain.Task)) {
	s.sendByExternalTransport = sendByExternalTransport
}

func (s *taskSender) Send() {
	if s.sendByExternalTransport == nil {
		panic("Transport for sending was not added")
	}

	for i := 0; i < 10; i++ {
		go s.confirm()
	}

	for {
		select {
		case task := <-s.chTasksReadyToSend:
			s.sendByExternalTransport(&task)
			s.chTasksToConfirm <- &task
		}
	}
}

func (s *taskSender) confirm() {

	var tasksToDelete []*domain.Task
	for task := range s.chTasksToConfirm {
		tasksToDelete = append(tasksToDelete, task)
		if len(tasksToDelete)%1e3 == 0 {
			if err := s.taskManager.ConfirmExecution(tasksToDelete); err != nil {
				panic(err)
			}
			tasksToDelete = nil
		}
	}

}
