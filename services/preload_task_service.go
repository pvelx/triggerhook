package services

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

func NewPreloadingTaskService(tm contracts.TaskManagerInterface) contracts.PreloadingTaskServiceInterface {
	return &preloadingTaskService{
		taskManager:     tm,
		chPreloadedTask: make(chan domain.Task, 1000000),
		timePreload:     5,
	}
}

type preloadingTaskService struct {
	taskManager           contracts.TaskManagerInterface
	chPreloadedTask       chan domain.Task
	timePreload           int64
	taskNumberInOneSearch int
}

func (s *preloadingTaskService) GetPreloadedChan() <-chan domain.Task {
	return s.chPreloadedTask
}

func (s *preloadingTaskService) AddNewTask(execTime int64) (*domain.Task, error) {
	task := domain.Task{
		ExecTime: execTime,
	}
	relativeTimeToExec := execTime - time.Now().Unix()

	isTaken := s.timePreload > relativeTimeToExec

	if err := s.taskManager.Create(&task, isTaken); err != nil {
		return nil, err
	}
	if isTaken {
		s.chPreloadedTask <- task
	}
	return &task, nil
}

func (s *preloadingTaskService) Preload() {
	countFails := 0

	for {
		countReadyToExec, err := s.taskManager.CountReadyToExec(s.timePreload)
		if err != nil {
			countFails++
			time.Sleep(time.Duration(s.timePreload) * 300 * time.Millisecond)

			if countFails == 10 {
				panic("Cannot get count of task for doing")
			}
			continue
		}
		countFails = 0

		if countReadyToExec > 0 {
			workers := (countReadyToExec / s.taskNumberInOneSearch) + 1
			for i := 0; i < workers; i++ {
				go func() {
					tasksToExec, err := s.taskManager.GetTasksBySecToExecTime(s.timePreload, s.taskNumberInOneSearch)
					if err != nil {
						panic("Cannot get tasks for doing")
						return
					}
					for _, task := range tasksToExec {
						s.chPreloadedTask <- task
					}
				}()
			}
		} else {
			time.Sleep(time.Duration(s.timePreload) * time.Second)
		}
	}
}
