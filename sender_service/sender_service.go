package sender_service

import (
	"time"

	"github.com/imdario/mergo"
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

type Options struct {
	BatchMaxItems            int
	BatchTimeout             time.Duration
	TasksToConfirmCap        int
	BatchesCap               int
	ConfirmationWorkersCount int
}

func New(
	taskManager contracts.TaskManagerInterface,
	tasksReadyToSend chan domain.Task,
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
		TasksToConfirmCap:        10000000,
		BatchesCap:               10000,
		ConfirmationWorkersCount: 5,
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

	return &senderService{
		taskManager:              taskManager,
		tasksToConfirm:           make(chan domain.Task, options.TasksToConfirmCap),
		tasksReadyToSend:         tasksReadyToSend,
		eh:                       eh,
		monitoring:               monitoring,
		batchesCap:               options.BatchesCap,
		confirmationWorkersCount: options.ConfirmationWorkersCount,
		batchTimeout:             options.BatchTimeout,
		batchMaxItems:            options.BatchMaxItems,
	}
}

type senderService struct {
	contracts.SenderServiceInterface
	tasksReadyToSend         chan domain.Task
	tasksToConfirm           chan domain.Task
	taskManager              contracts.TaskManagerInterface
	eh                       contracts.EventHandlerInterface
	monitoring               contracts.MonitoringInterface
	batchTimeout             time.Duration
	confirmationWorkersCount int
	batchMaxItems            int
	batchesCap               int
}

func (s *senderService) Run() {
	batchTasks := s.generateBatch(s.tasksToConfirm)

	for w := 0; w < s.confirmationWorkersCount; w++ {
		go func() {
			for batch := range batchTasks {
				if err := s.taskManager.ConfirmExecution(batch); err != nil {
					s.eh.New(contracts.LevelFatal, err.Error(), nil)
				}

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
		}()
	}
}

func (s *senderService) generateBatch(tasks <-chan domain.Task) chan []domain.Task {
	batches := make(chan []domain.Task, s.batchesCap)

	go func() {
		for {
			var batch []domain.Task
			expire := time.NewTimer(s.batchTimeout)
			for {
				select {
				case value := <-tasks:
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
				batches <- batch

				if err := s.monitoring.Publish(contracts.WaitingForConfirmation, int64(len(batch))); err != nil {
					s.eh.New(contracts.LevelError, err.Error(), nil)
				}
				if len(batches) == cap(batches) {
					s.eh.New(contracts.LevelError, "channel is full", nil)
				}
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
	return &taskToSend{
		monitoring: s.monitoring,
		eh:         s.eh,
		rollback:   s.tasksReadyToSend,
		confirm:    s.tasksToConfirm,
		task:       <-s.tasksReadyToSend,
	}
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
	}
}

func (tts *taskToSend) Rollback() {
	if !tts.isProcessed {
		tts.isProcessed = true
		tts.rollback <- tts.task
	}
}
