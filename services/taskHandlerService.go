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
	service := &taskHandlerService{taskService: taskService}
	service.construct()
	return service
}

type taskHandlerService struct {
	taskService     tasks.Service
	chTaskToExecute chan tasks.Task
}

func (s *taskHandlerService) construct() {
	s.chTaskToExecute = make(chan tasks.Task, 1000000)
}

func (s *taskHandlerService) Create(task tasks.Task) {
	s.taskService.Create(task)
	s.chTaskToExecute <- task
}

func (s *taskHandlerService) Delete(tasks.Task) {

	panic("implement me")
}

func (s *taskHandlerService) findToExecMock() {

	now := time.Now().Unix()
	var idx int64 = 0
	countOfTasks := int64(2e+7)
	for i := now - countOfTasks/2; i < now+countOfTasks/2; i = i + 1 {
		s.chTaskToExecute <- tasks.Task{Id: idx, ExecTime: i}
		idx++
	}

	for {
		ts := time.Now().Unix()
		con := int64(40)
		idx++
		s.chTaskToExecute <- tasks.Task{Id: idx, ExecTime: ts, TakenByConnection: &con}
		time.Sleep(time.Second)
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
		if task.Id%1e+6 == 0 {
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

func (s *taskHandlerService) addTaskToQueue(queue *PriorityQueue, updatedQueue chan bool, mut *sync.Mutex) {
	for {
		select {
		case task := <-s.chTaskToExecute:
			mut.Lock()
			queue.AddTask(&task)
			mut.Unlock()
			updatedQueue <- true
		}
	}
}

func (s *taskHandlerService) Execute() {
	chExport := make(chan tasks.Task)

	//initData := s.getData(5)
	queue := NewQueue([]tasks.Task{})

	updatedQueue := make(chan bool)
	mut := &sync.Mutex{}

	go s.findToExecMock()
	go s.send(chExport)
	//go s.findToExec(ch, queue)

	go s.addTaskToQueue(&queue, updatedQueue, mut)

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
