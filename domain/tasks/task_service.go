package tasks

import (
	"errors"
	"github.com/VladislavPav/trigger-hook/utils"
	"time"
)

type RepositoryInterface interface {
	Create(task *Task, isTaken bool) *utils.ErrorRepo
	FindBySecToExecTime(secToNow int64) (Tasks, *utils.ErrorRepo)
	ChangeStatusToCompleted(*Task) *utils.ErrorRepo
}

type Service interface {
	Create(execTime int64) (*Task, *error)
	Preload()
}

func NewService(repo RepositoryInterface, chPreloadedTask chan Task) Service {
	return &service{
		repo:            repo,
		chPreloadedTask: chPreloadedTask,
		timePreload:     5,
	}
}

type service struct {
	repo            RepositoryInterface
	chPreloadedTask chan Task
	timePreload     int64
}

func (s *service) Create(execTime int64) (*Task, *error) {
	task := Task{
		ExecTime: execTime,
	}
	relativeTimeToExec := execTime - time.Now().Unix()
	isTaken := false
	if s.timePreload > relativeTimeToExec {
		isTaken = true
	}
	if err := s.repo.Create(&task, isTaken); err != nil {
		err := errors.New(err.Error())
		return nil, &err
	}
	if isTaken {
		s.chPreloadedTask <- task
	}
	return &task, nil
}

func (s *service) findToExecMock() {
	now := time.Now().Unix()
	var idx int64 = 0
	countOfTasks := int64(2e+7)
	for i := now - countOfTasks/2; i < now+countOfTasks/2; i = i + 1 {
		s.chPreloadedTask <- Task{Id: idx, ExecTime: i}
		idx++
	}

	for {
		ts := time.Now().Unix()
		con := int64(40)
		idx++
		s.chPreloadedTask <- Task{Id: idx, ExecTime: ts, TakenByConnection: &con}
		time.Sleep(time.Second)
	}
}

func (s *service) Preload() {
	go s.findToExecMock()

	//for {
	//	tasksToExec, err := s.repo.FindBySecToExecTime(s.timePreload / 2)
	//	if err != nil {
	//		panic(err)
	//	}
	//	for _, task := range tasksToExec {
	//		s.chPreloadedTask <- task
	//	}
	//
	//	time.Sleep(time.Duration(s.timePreload) * time.Second)
	//}
}
