package events

import "github.com/VladislavPav/trigger-hook/domain/tasks"

//пишется после выполнения. лог
type Event struct {
	Id          int64
	CompletedAt string
	Task        tasks.Task
}
