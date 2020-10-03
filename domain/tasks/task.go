package tasks

type Status int

const StatusAwaiting Status = 1
const StatusCompleted Status = 2

type Task struct {
	Id                int64  `json:"id"`
	ExecTime          int64  `json:"exec_time"`           //Time of execution a task
	TakenByConnection *int64 `json:"taken_by_connection"` //Which connection take a task
	Status            Status `json:"status"`
	DeletedAt         int64  `json:"deleted_at"`
}

type Tasks []Task
