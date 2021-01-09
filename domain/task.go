package domain

type Task struct {
	Id       string `json:"id"`
	ExecTime int64  `json:"exec_time"` //Time of execution of a task
}
