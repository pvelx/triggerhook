package waiting_service

import (
	"context"
	"math"
	"time"

	"github.com/imdario/mergo"
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

/*	--------------------------------------------------
	Prioritized task list
*/

type prioritizedTaskListInterface interface {
	/*
		Count of tasks in the list
	*/
	Len() int

	/*
		Add a task to the list based on priority
	*/
	Add(task domain.Task)

	/*
		Take the most prioritized task and delete from list
	*/
	Take() *domain.Task

	/*
		Searches for a task and deletes it return true when the task was deleted, false - when was not found
	*/
	DeleteIfExist(taskId string) bool
}

type Options struct {
	TasksReadyToSendCap   int //Deprecated
	CanceledTasksCap      int //Deprecated
	GreedyProcessingLimit int
}

func New(
	preloadedTasks <-chan domain.Task,
	monitoring contracts.MonitoringInterface,
	taskManager contracts.TaskManagerInterface,
	eventHandler contracts.EventHandlerInterface,
	options *Options,
) contracts.WaitingServiceInterface {

	if options == nil {
		options = &Options{}
	}

	if err := mergo.Merge(options, Options{
		GreedyProcessingLimit: 10,
	}); err != nil {
		panic(err)
	}

	tasksWaitingList := NewPrioritizedTask([]domain.Task{})

	if err := monitoring.Listen(contracts.Preloaded, func() int64 {
		return int64(tasksWaitingList.Len())
	}); err != nil {
		panic(err)
	}
	if err := monitoring.Init(contracts.DeletingRate, contracts.VelocityMetricType); err != nil {
		panic(err)
	}

	service := &waitingService{
		tasksWaitingList:      tasksWaitingList,
		preloadedTasks:        preloadedTasks,
		canceledTasks:         make(chan string, 1),
		tasksReadyToSend:      make(chan domain.Task, 1),
		greedyProcessingLimit: options.GreedyProcessingLimit,
		monitoring:            monitoring,
		taskManager:           taskManager,
		eh:                    eventHandler,
	}

	return service
}

type waitingService struct {
	tasksWaitingList      prioritizedTaskListInterface
	preloadedTasks        <-chan domain.Task
	canceledTasks         chan string
	tasksReadyToSend      chan domain.Task
	greedyProcessingLimit int
	monitoring            contracts.MonitoringInterface
	taskManager           contracts.TaskManagerInterface
	eh                    contracts.EventHandlerInterface
}

func (s *waitingService) GetReadyToSendChan() chan domain.Task {
	return s.tasksReadyToSend
}

func (s *waitingService) CancelIfExist(ctx context.Context, taskId string) error {
	if err := s.taskManager.Delete(ctx, taskId); err != nil {
		return err
	}
	s.canceledTasks <- taskId

	if err := s.monitoring.Publish(contracts.DeletingRate, 1); err != nil {
		s.eh.New(contracts.LevelError, err.Error(), nil)
	}

	return nil
}

func (s *waitingService) Run() {
	var sleep time.Duration
	var task *domain.Task
	for {
		sleep = time.Duration(math.MaxInt64)
		task = s.tasksWaitingList.Take()

		if task != nil {
			sleep = time.Duration(task.ExecTime-time.Now().Unix()) * time.Second
		}

		if sleep > 0 {
			t := time.NewTimer(sleep)
			select {
			case <-t.C:
			case newTask := <-s.preloadedTasks:
				t.Stop()
				if task != nil {
					s.tasksWaitingList.Add(*task)
				}
				s.tasksWaitingList.Add(newTask)

				//	Adding tasks to the waiting list in this block will reduce the chance of blocking
				//	sending tasks to tasksReadyToSend because we do this during hibernation
				//
				// 	The use of greedy processing allows you to reduce the number of workings of the external cycle,
				//	reduce the number of operations
				for i, empty := 0, false; i < s.greedyProcessingLimit && !empty; i++ {
					select {
					case task := <-s.preloadedTasks:
						s.tasksWaitingList.Add(task)
					default:
						empty = true
					}
				}

				continue
			case taskId := <-s.canceledTasks:
				t.Stop()
				if task != nil && taskId != task.Id {
					s.tasksWaitingList.Add(*task)
				}
				s.tasksWaitingList.DeleteIfExist(taskId)

				//	Same as for the preloadedTasks block
				for i, empty := 0, false; i < s.greedyProcessingLimit && !empty; i++ {
					select {
					case taskId := <-s.canceledTasks:
						s.tasksWaitingList.DeleteIfExist(taskId)
					default:
						empty = true
					}
				}

				continue
			}
		}

		s.tasksReadyToSend <- *task
	}
}
