package services

import (
	"fmt"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
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

func findToExecMock(key *[]int64, queue map[int64][]tasks.Task, mut *sync.Mutex) {
	now := time.Now().Unix()
	var idx int64 = 0
	countOfTasks := int64(16000000)
	for i := now - countOfTasks/2; i < now+countOfTasks/2; i++ {
		idx++

		mut.Lock()
		_, ok := queue[i]
		if !ok {
			queue[i] = []tasks.Task{{Id: idx, ExecTime: i}}
		}
		*key = append(*key, i)
		//sort.Slice(key, func(i int, j int) bool {
		//	return key[i] < key[j]
		//})
		mut.Unlock()
	}

}

func (s *taskHandlerService) findToExec(ch chan tasks.Task, queue map[int64][]tasks.Task) {
	var periodicity int64 = 5
	for {
		tasksToExec, err := s.taskService.FindToExec(periodicity * (10 / 8))
		if err != nil {
			fmt.Println(err)
		}
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

func (s *taskHandlerService) Execute() {
	var key []int64
	newTaskCh := make(chan tasks.Task)
	mut := &sync.Mutex{}
	var queue = map[int64][]tasks.Task{}

	chExport := make(chan tasks.Task)
	go findToExecMock(&key, queue, mut)
	go s.send(chExport)
	//go s.findToExec(ch, queue)

LOOP:
	for {
		var execTime int64
		var sleepTime int64
		if len(key) == 0 {
			sleepTime = 1e+9
		} else {
			execTime := key[0]
			sleepTime = (execTime * 1e+9) - time.Now().UnixNano()
		}

		if sleepTime > 0 {
			timer := time.NewTimer(time.Duration(sleepTime) * time.Nanosecond)
			fmt.Println("timer start")
			select {
			case <-timer.C:
				break
			case task := <-newTaskCh:
				if !timer.Stop() {
					<-timer.C
				}
				fmt.Println("added task", task)
				break LOOP
			}
			fmt.Println("timer end")
			//time.Sleep(time.Duration(sleepTime) * time.Nanosecond)

		}

		if len(key) > 0 {
			execTime = key[0]

			mut.Lock()
			for _, task := range queue[execTime] {
				chExport <- task
			}
			key = key[1:]
			delete(queue, execTime)
			mut.Unlock()
			//fmt.Println("queue count:", len(queue))
		} else {
			fmt.Println("key=0", key)
		}
	}
}
