package structures

import "github.com/VladislavPav/trigger-hook/domain/tasks"

//TasksWaitingList
type TaskQueueInterface interface {
	Offer(task *tasks.Task)
	Poll() *tasks.Task
}
