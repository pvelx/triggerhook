package sender_service

import (
	"context"
	"sync"
	"time"

	"github.com/imdario/mergo"
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

type Options struct {
	BatchMaxItems            int
	BatchTimeout             time.Duration
	TasksToConfirmCap        int //Deprecated
	BatchesCap               int //Deprecated
	ConfirmationWorkersCount int
	CtxTimeout               time.Duration
}

var taskToSendPool sync.Pool

func New(
	taskManager contracts.TaskManagerInterface,
	tasksReadyToSend <-chan domain.Task,
	eh contracts.EventHandlerInterface,
	monitoring contracts.MonitoringInterface,
	options *Options,
) contracts.SenderServiceInterface {

	if options == nil {
		options = &Options{}
	}

	if err := mergo.Merge(options, Options{
		BatchMaxItems:            1000,
		BatchTimeout:             50 * time.Millisecond,
		ConfirmationWorkersCount: 5,
		CtxTimeout:               5 * time.Second,
	}); err != nil {
		panic(err)
	}

	err := monitoring.Init(contracts.ConfirmationRate, contracts.VelocityMetricType)
	if err != nil {
		panic(err)
	}
	if err := monitoring.Init(contracts.SendingRate, contracts.VelocityMetricType); err != nil {
		panic(err)
	}
	if err := monitoring.Init(contracts.WaitingForConfirmation, contracts.IntegralMetricType); err != nil {
		panic(err)
	}

	tasksToConfirm := make(chan domain.Task, options.BatchMaxItems)
	buffer := NewBuffer()

	taskToSendPool = sync.Pool{
		New: func() interface{} {
			return &taskToSend{
				monitoring: monitoring,
				eh:         eh,
				rollback:   buffer.In,
				confirm:    tasksToConfirm,
			}
		},
	}

	return &senderService{
		taskManager:              taskManager,
		tasksToConfirm:           tasksToConfirm,
		tasksReadyToSend:         tasksReadyToSend,
		eh:                       eh,
		monitoring:               monitoring,
		confirmationWorkersCount: options.ConfirmationWorkersCount,
		batchTimeout:             options.BatchTimeout,
		batchMaxItems:            options.BatchMaxItems,
		taskBuffer:               buffer,
		ctxTimeout:               options.CtxTimeout,
	}
}

type senderService struct {
	contracts.SenderServiceInterface
	tasksReadyToSend         <-chan domain.Task
	tasksToConfirm           chan domain.Task
	taskManager              contracts.TaskManagerInterface
	eh                       contracts.EventHandlerInterface
	monitoring               contracts.MonitoringInterface
	batchTimeout             time.Duration
	confirmationWorkersCount int
	batchMaxItems            int
	taskBuffer               *buffer
	ctxTimeout               time.Duration
}

func (s *senderService) Run() {
	batchTasks := s.generateBatch(s.tasksToConfirm)

	for w := 0; w < s.confirmationWorkersCount; w++ {
		go s.confirmation(batchTasks)
	}
}

func (s *senderService) confirmation(batchTasks chan []domain.Task) {
	for batch := range batchTasks {
		ctx, stop := context.WithTimeout(context.Background(), s.ctxTimeout)
		if err := s.taskManager.ConfirmExecution(ctx, batch); err != nil {
			s.eh.New(contracts.LevelFatal, err.Error(), nil)
		}
		stop()

		s.eh.New(contracts.LevelDebug, "confirmed tasks", map[string]interface{}{
			"count of task": len(batch),
		})

		if err := s.monitoring.Publish(contracts.WaitingForConfirmation, int64(-len(batch))); err != nil {
			s.eh.New(contracts.LevelError, err.Error(), nil)
		}
		if err := s.monitoring.Publish(contracts.ConfirmationRate, int64(len(batch))); err != nil {
			s.eh.New(contracts.LevelError, err.Error(), nil)
		}
	}
}

func (s *senderService) generateBatch(tasks <-chan domain.Task) chan []domain.Task {
	batches := make(chan []domain.Task, 1)
	updateQueue := make(chan bool, 1)
	queue := &batchTaskQueue{}

	go func() {
		defer close(updateQueue)
		for {
			batch := make([]domain.Task, 0, s.batchMaxItems)
			expire := time.NewTimer(s.batchTimeout)
			for {
				select {
				case value, ok := <-tasks:
					if !ok {
						return
					}
					batch = append(batch, value)
					if len(batch) == s.batchMaxItems {
						expire.Stop()
						goto done
					}

				case <-expire.C:
					goto done
				}
			}

		done:
			if len(batch) > 0 {
				queue.Push(batch)

				if len(updateQueue) == 0 {
					updateQueue <- true
				}

				if err := s.monitoring.Publish(contracts.WaitingForConfirmation, int64(len(batch))); err != nil {
					s.eh.New(contracts.LevelError, err.Error(), nil)
				}
			}
		}
	}()

	go func() {
		defer close(batches)
		for {
			if tasks, err := queue.Pop(); err == QueueIsEmpty {

				//	Exit when the queue is emptied
				//	and when its update stops
				if _, ok := <-updateQueue; !ok {
					return
				}
			} else {
				batches <- tasks
			}
		}
	}()

	return batches
}

type taskToSend struct {
	monitoring  contracts.MonitoringInterface
	eh          contracts.EventHandlerInterface
	isProcessed bool
	confirm     chan domain.Task
	rollback    chan domain.Task
	task        domain.Task
}

func (s *senderService) Consume() contracts.TaskToSendInterface {
	taskToSend := taskToSendPool.Get().(*taskToSend)
	taskToSend.isProcessed = false

	select {
	case taskToSend.task = <-s.tasksReadyToSend:
	case taskToSend.task = <-s.taskBuffer.Out:
	}

	return taskToSend
}

func (tts *taskToSend) Task() domain.Task {
	return tts.task
}

func (tts *taskToSend) Confirm() {
	if !tts.isProcessed {
		if err := tts.monitoring.Publish(contracts.SendingRate, 1); err != nil {
			tts.eh.New(contracts.LevelError, err.Error(), nil)
		}

		tts.isProcessed = true
		tts.confirm <- tts.task
		taskToSendPool.Put(tts)
	}
}

func (tts *taskToSend) Rollback() {
	if !tts.isProcessed {
		tts.isProcessed = true
		tts.rollback <- tts.task
		taskToSendPool.Put(tts)
	}
}
