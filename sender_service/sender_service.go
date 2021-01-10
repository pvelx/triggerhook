package sender_service

import (
	"github.com/imdario/mergo"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

type Options struct {
	/*
		Setting the function with the desired message sending method (for example, RabbitMQ)
	*/
	Transport                func(task domain.Task)
	BatchMaxItems            int
	BatchTimeout             time.Duration
	ChTasksToConfirmCap      int
	ChBatchesCap             int
	ConfirmationWorkersCount int
}

func New(
	taskManager contracts.TaskManagerInterface,
	chTasksReadyToSend <-chan domain.Task,
	eeh contracts.EventErrorHandlerInterface,
	monitoring contracts.MonitoringInterface,
	options Options,
) contracts.TaskSenderInterface {

	if options.Transport == nil {
		panic("You have to define a transport for sending tasks")
	}

	if err := mergo.Merge(&options, Options{
		BatchMaxItems:            1000,
		BatchTimeout:             50 * time.Millisecond,
		ChTasksToConfirmCap:      10000000,
		ChBatchesCap:             10000,
		ConfirmationWorkersCount: 5,
	}); err != nil {
		panic(err)
	}

	if err := monitoring.Init(contracts.SpeedOfConfirmation, contracts.VelocityMetricType); err != nil {
		panic(err)
	}
	if err := monitoring.Init(contracts.SpeedOfSending, contracts.VelocityMetricType); err != nil {
		panic(err)
	}
	if err := monitoring.Init(contracts.WaitingForConfirmation, contracts.IntegralMetricType); err != nil {
		panic(err)
	}

	return &taskSender{
		sendByExternalTransport:  options.Transport,
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
	sendByExternalTransport  func(task domain.Task)
	chTasksReadyToSend       <-chan domain.Task
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
	if s.sendByExternalTransport == nil {
		panic("Transport for sending was not added")
	}
	batchTasksCh := s.generateBatch(s.chTasksToConfirm, s.batchMaxItems, s.batchTimeout)

	go s.confirmBatch(batchTasksCh)

	for task := range s.chTasksReadyToSend {
		s.sendByExternalTransport(task)
		s.chTasksToConfirm <- task
		if err := s.monitoring.Publish(contracts.SpeedOfSending, 1); err != nil {
			s.eeh.New(contracts.LevelError, err.Error(), nil)
		}
	}
}

func (s *taskSender) confirmBatch(batchTasksCh chan []domain.Task) {
	for i := 0; i < s.confirmationWorkersCount; i++ {
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

func (s *taskSender) generateBatch(tasks <-chan domain.Task, maxItems int, maxTimeout time.Duration) chan []domain.Task {
	batches := make(chan []domain.Task, s.chBatchesCap)

	go func() {
		for {
			var batch []domain.Task
			expire := time.After(maxTimeout)
			for {
				select {
				case value := <-tasks:
					batch = append(batch, value)
					if len(batch) == maxItems {
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
