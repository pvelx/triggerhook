package sender_service

import (
	"github.com/imdario/mergo"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

type Options struct {
	BatchMaxItems            int
	BatchTimeout             time.Duration
	ChTasksToConfirmCap      int
	ChBatchesCap             int
	ConfirmationWorkersCount int
}

func New(
	taskManager contracts.TaskManagerInterface,
	chTasksReadyToSend chan domain.Task,
	eeh contracts.EventErrorHandlerInterface,
	monitoring contracts.MonitoringInterface,
	options *Options,
) contracts.TaskSenderInterface {

	if options == nil {
		options = &Options{}
	}

	if err := mergo.Merge(options, Options{
		BatchMaxItems:            1000,
		BatchTimeout:             50 * time.Millisecond,
		ChTasksToConfirmCap:      10000000,
		ChBatchesCap:             10000,
		ConfirmationWorkersCount: 5,
	}); err != nil {
		panic(err)
	}

	err := monitoring.Init(contracts.SpeedOfConfirmation, contracts.VelocityMetricType)
	if err != nil {
		panic(err)
	}
	if err := monitoring.Init(contracts.SpeedOfSending, contracts.VelocityMetricType); err != nil {
		panic(err)
	}
	if err := monitoring.Init(contracts.WaitingForConfirmation, contracts.IntegralMetricType); err != nil {
		panic(err)
	}

	return &taskSender{
		taskManager:              taskManager,
		chTasksToConfirm:         make(chan domain.Task, options.ChTasksToConfirmCap),
		chTasksReadyToSend:       chTasksReadyToSend,
		eeh:                      eeh,
		monitoring:               monitoring,
		chBatchesCap:             options.ChBatchesCap,
		confirmationWorkersCount: options.ConfirmationWorkersCount,
		batchTimeout:             options.BatchTimeout,
		batchMaxItems:            options.BatchMaxItems,
	}
}

type taskSender struct {
	contracts.TaskSenderInterface
	chTasksReadyToSend       chan domain.Task
	chTasksToConfirm         chan domain.Task
	taskManager              contracts.TaskManagerInterface
	eeh                      contracts.EventErrorHandlerInterface
	monitoring               contracts.MonitoringInterface
	batchTimeout             time.Duration
	confirmationWorkersCount int
	batchMaxItems            int
	chBatchesCap             int
}

func (s *taskSender) Run() {
	batchTasksCh := s.generateBatch(s.chTasksToConfirm)

	for w := 0; w < s.confirmationWorkersCount; w++ {
		go func() {
			for batch := range batchTasksCh {
				if err := s.taskManager.ConfirmExecution(batch); err != nil {
					s.eeh.New(contracts.LevelFatal, err.Error(), nil)
				}

				s.eeh.New(contracts.LevelDebug, "confirmed tasks", map[string]interface{}{
					"count of task": len(batch),
				})

				if err := s.monitoring.Publish(contracts.WaitingForConfirmation, int64(-len(batch))); err != nil {
					s.eeh.New(contracts.LevelError, err.Error(), nil)
				}
				if err := s.monitoring.Publish(contracts.SpeedOfConfirmation, int64(len(batch))); err != nil {
					s.eeh.New(contracts.LevelError, err.Error(), nil)
				}
			}
		}()
	}
}

func (s *taskSender) Consume() contracts.TaskToSendInterface {
	return &taskToSend{
		monitoring: s.monitoring,
		eeh:        s.eeh,
		chRollback: s.chTasksReadyToSend,
		chConfirm:  s.chTasksToConfirm,
		task:       <-s.chTasksReadyToSend,
	}
}

func (s *taskSender) generateBatch(tasks <-chan domain.Task) chan []domain.Task {
	batches := make(chan []domain.Task, s.chBatchesCap)

	go func() {
		for {
			var batch []domain.Task
			expire := time.After(s.batchTimeout)
			for {
				select {
				case value := <-tasks:
					batch = append(batch, value)
					if len(batch) == s.batchMaxItems {
						goto done
					}

				case <-expire:
					goto done
				}
			}

		done:
			if len(batch) > 0 {
				batches <- batch

				if err := s.monitoring.Publish(contracts.WaitingForConfirmation, int64(len(batch))); err != nil {
					s.eeh.New(contracts.LevelError, err.Error(), nil)
				}
				if len(batches) == cap(batches) {
					s.eeh.New(contracts.LevelError, "channel is full", nil)
				}
			}
		}
	}()

	return batches
}

type taskToSend struct {
	monitoring  contracts.MonitoringInterface
	eeh         contracts.EventErrorHandlerInterface
	isProcessed bool
	chConfirm   chan domain.Task
	chRollback  chan domain.Task
	task        domain.Task
}

func (tts *taskToSend) Task() domain.Task {
	return tts.task
}

func (tts *taskToSend) Confirm() {
	if !tts.isProcessed {
		if err := tts.monitoring.Publish(contracts.SpeedOfSending, 1); err != nil {
			tts.eeh.New(contracts.LevelError, err.Error(), nil)
		}

		tts.isProcessed = true
		tts.chConfirm <- tts.task
	}
}

func (tts *taskToSend) Rollback() {
	if !tts.isProcessed {
		tts.isProcessed = true
		tts.chRollback <- tts.task
	}
}
