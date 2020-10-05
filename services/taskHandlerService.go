package services

import (
	"fmt"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"math"
	"sync"
	"time"
)

type TaskHandlerServiceInterface interface {
	Create(tasks.Task)
	Delete(tasks.Task)
	Execute()
}

func NewTaskHandlerService(taskService tasks.Service) TaskHandlerServiceInterface {
	return &taskHandlerService{taskService: taskService}
}

type taskHandlerService struct {
	taskService tasks.Service
}

func (s *taskHandlerService) Create(task tasks.Task) {
	s.taskService.Create(task)
}

func (s *taskHandlerService) Delete(tasks.Task) {

	panic("implement me")
}

func findToExecMock(queue *PriorityQueue, mut *sync.Mutex, updatedQueue chan bool) {
	now := time.Now().Unix()
	var idx int64 = 0
	countOfTasks := int64(20000000)
	for i := now - countOfTasks/2; i < now+countOfTasks/2; i = i + 1 {
		idx++
		mut.Lock()
		queue.AddTask(&tasks.Task{Id: idx, ExecTime: i})
		mut.Unlock()
	}
	updatedQueue <- true

	for {
		mut.Lock()
		ts := time.Now().Unix()
		con := int64(40)
		idx++
		queue.AddTask(&tasks.Task{Id: idx, ExecTime: ts, TakenByConnection: &con})
		mut.Unlock()
		time.Sleep(time.Second)
		updatedQueue <- true
	}
}

func (s *taskHandlerService) findToExec(ch chan tasks.Task) {
	var periodicity int64 = 5
	for {
		tasksToExec := s.getData(periodicity * (10 / 8))
		for _, task := range tasksToExec {
			ch <- task
		}

		time.Sleep(time.Duration(periodicity) * time.Second)
	}
}

func (s *taskHandlerService) send(chExport chan tasks.Task) {
	for task := range chExport {
		if task.Id%1000000 == 0 {
			fmt.Println("Send:", task)
		}
		//if err != s.repo.ChangeStatusToCompleted(tasks) {
		//	return nil, errors.New(err.Error())
		//}
	}
}

func (s *taskHandlerService) getData(sec int64) []tasks.Task {
	tasksToExec, err := s.taskService.FindToExec(sec)
	if err != nil {
		fmt.Println(err)
	}
	return tasksToExec
}

func (s *taskHandlerService) Execute() {

	//initData := s.getData(5)
	queue := NewQueue([]tasks.Task{})

	updatedQueue := make(chan bool)
	mut := &sync.Mutex{}

	chExport := make(chan tasks.Task)
	go findToExecMock(&queue, mut, updatedQueue)
	go s.send(chExport)
	//go s.findToExec(ch, queue)

	for {
		var sleepTime int64
		var task *tasks.Task

		mut.Lock()
		task = queue.GetTask()
		mut.Unlock()

		if task == nil {
			sleepTime = math.MaxInt64
		} else {
			sleepTime = (task.ExecTime * 1e+9) - time.Now().UnixNano()
		}

		if sleepTime > 0 {
			timer := time.NewTimer(time.Duration(sleepTime) * time.Nanosecond)
			for len(updatedQueue) > 0 {
				<-updatedQueue
			}
			select {
			case <-timer.C:
				break
			case <-updatedQueue:
				if !timer.Stop() {
					<-timer.C
				}
				if task != nil {
					mut.Lock()
					queue.AddTask(task)
					mut.Unlock()
				}

				continue
			}
		}

		chExport <- *task
	}
}
