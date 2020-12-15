package domain

type Task struct {
	Id       int64 `json:"id"`
	ExecTime int64 `json:"exec_time"` //Time of execution a task
}

type Tasks []Task
