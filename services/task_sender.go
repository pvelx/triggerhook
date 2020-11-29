package services

import (
	"fmt"
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
)

type taskSender struct {
	contracts.TaskSenderInterface
	chTasksReadyToSend chan tasks.Task
}

func NewTaskSender(chTasksReadyToSend chan tasks.Task) contracts.TaskSenderInterface {
	return &taskSender{
		chTasksReadyToSend: chTasksReadyToSend,
	}
}

func (s *taskSender) Send() {
	for task := range s.chTasksReadyToSend {
		if task.Id%1e+6 == 0 {
			fmt.Println("Send:", task)
		}
		//if err != s.repo.ChangeStatusToCompleted(tasks) {
		//	return nil, errors.New(err.Error())
		//}
	}
}
