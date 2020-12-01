package services

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

func NewPreloadingTaskService(tm contracts.TaskManagerInterface, chPreloadedTask chan<- domain.Task) contracts.PreloadingTaskServiceInterface {
	return &preloadingTaskService{
		taskManager:     tm,
		chPreloadedTask: chPreloadedTask,
		timePreload:     5,
	}
}

type preloadingTaskService struct {
	taskManager     contracts.TaskManagerInterface
	chPreloadedTask chan<- domain.Task
	timePreload     int64
}

func (s *preloadingTaskService) AddNewTask(execTime int64) (*domain.Task, *error) {
	task := domain.Task{
		ExecTime: execTime,
	}
	relativeTimeToExec := execTime - time.Now().Unix()
	isTaken := false
	if s.timePreload > relativeTimeToExec {
		isTaken = true
	}
	if err := s.taskManager.Create(&task, isTaken); err != nil {
		return nil, err
	}
	if isTaken {
		s.chPreloadedTask <- task
	}
	return &task, nil
}

func (s *preloadingTaskService) findToExecMock() {
	now := time.Now().Unix()
	var idx int64 = 0
	countOfTasks := int64(2e+7)
	for i := now - countOfTasks/2; i < now+countOfTasks/2; i = i + 1 {
		s.chPreloadedTask <- domain.Task{Id: idx, ExecTime: i}
		idx++
	}

	for {
		ts := time.Now().Unix()
		con := int64(40)
		idx++
		s.chPreloadedTask <- domain.Task{Id: idx, ExecTime: ts, TakenByConnection: &con}
		time.Sleep(time.Second)
	}
}

func (s *preloadingTaskService) Preload() {
	go s.findToExecMock()

	for {
		tasksToExec := s.taskManager.GetTasksBySecToExecTime(s.timePreload)
		for _, task := range tasksToExec {
			s.chPreloadedTask <- task
		}

		time.Sleep(time.Duration(s.timePreload) * time.Second)
	}
}
