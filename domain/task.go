package domain

type Task struct {
	Id       string `json:"id"`        //Uuid of the task. If not specified it will be created automatically
	ExecTime int64  `json:"exec_time"` //Time of execution of the task. Required parameter
}
