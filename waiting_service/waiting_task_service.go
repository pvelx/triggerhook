package waiting_service

import (
	"github.com/imdario/mergo"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/prioritized_task_list"
	"math"
	"sync"
	"time"
)

type Options struct {
	ChTasksReadyToSendCap int
	ChCanceledTasksCap    int
}

func New(
	chPreloadedTasks <-chan domain.Task,
	monitoring contracts.MonitoringInterface,
	taskManager contracts.TaskManagerInterface,
	eventHandler contracts.EventErrorHandlerInterface,
	options *Options,
) contracts.WaitingTaskServiceInterface {

	if options == nil {
		options = &Options{}
	}

	if err := mergo.Merge(options, Options{
		ChTasksReadyToSendCap: 1000000,
		ChCanceledTasksCap:    1000000,
	}); err != nil {
		panic(err)
	}

	tasksWaitingList := prioritized_task_list.New([]domain.Task{})

	if err := monitoring.Listen(contracts.Preloaded, tasksWaitingList.Len()); err != nil {
		panic(err)
	}
	if err := monitoring.Init(contracts.SpeedOfDeleting, contracts.VelocityMetricType); err != nil {
		panic(err)
	}

	service := &waitingTaskService{
		tasksWaitingList:   tasksWaitingList,
		chPreloadedTasks:   chPreloadedTasks,
		chCanceledTasks:    make(chan string, options.ChCanceledTasksCap),
		chTasksReadyToSend: make(chan domain.Task, options.ChTasksReadyToSendCap),
		mu:                 &sync.Mutex{},
		monitoring:         monitoring,
		taskManager:        taskManager,
		eeh:                eventHandler,
	}

	return service
}

type waitingTaskService struct {
	tasksWaitingList   contracts.PrioritizedTaskListInterface
	chPreloadedTasks   <-chan domain.Task
	chCanceledTasks    chan string
	chTasksReadyToSend chan domain.Task
	mu                 *sync.Mutex
	monitoring         contracts.MonitoringInterface
	taskManager        contracts.TaskManagerInterface
	eeh                contracts.EventErrorHandlerInterface
}

func (s *waitingTaskService) GetReadyToSendChan() <-chan domain.Task {
	return s.chTasksReadyToSend
}

func (s *waitingTaskService) deleteTaskFromWaitingList(taskId string) {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	s.tasksWaitingList.DeleteIfExist(taskId)
}

func (s *waitingTaskService) addTaskToWaitingList(task domain.Task) {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	s.tasksWaitingList.Add(task)
}

func (s *waitingTaskService) takeTaskFromWaitingList() *domain.Task {
	s.mu.Lock()
	defer func() { s.mu.Unlock() }()
	return s.tasksWaitingList.Take()
}

func (s *waitingTaskService) CancelIfExist(taskId string) error {
	if err := s.taskManager.Delete(taskId); err != nil {
		return err
	}
	s.chCanceledTasks <- taskId

	if err := s.monitoring.Publish(contracts.SpeedOfDeleting, 1); err != nil {
		s.eeh.New(contracts.LevelError, err.Error(), nil)
	}

	return nil
}

func (s *waitingTaskService) Run() {
	updatedQueue := make(chan bool)

	go func() {
		for {
			select {
			case taskId, ok := <-s.chCanceledTasks:
				if ok {
					s.deleteTaskFromWaitingList(taskId)
					updatedQueue <- true
				} else {
					panic("chan was closed")
				}
			case task, ok := <-s.chPreloadedTasks:
				if ok {
					s.addTaskToWaitingList(task)
					updatedQueue <- true
				} else {
					panic("chan was closed")
				}
			}
		}
	}()

	var sleepTime int64

	for {
		task := s.takeTaskFromWaitingList()

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
			case <-updatedQueue:
				if !timer.Stop() {
					<-timer.C
				}
				if task != nil {
					s.addTaskToWaitingList(*task)
				}

				continue
			}
		}

		s.chTasksReadyToSend <- *task
	}
}
