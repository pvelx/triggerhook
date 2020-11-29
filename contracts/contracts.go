package contracts

import "github.com/VladislavPav/trigger-hook/domain/tasks"

type TasksWaitingListInterface interface {
	Add(task *tasks.Task)
	Take() *tasks.Task
}

type TaskSenderInterface interface {
	Send()
}
