package services

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"time"
)

type Options struct {
	batchMaxItems       int
	batchTimeout        time.Duration
	chTasksToConfirmLen int
	chBatchesLen        int
}

func NewTaskSender(
	taskManager contracts.TaskManagerInterface,
	chTasksReadyToSend <-chan domain.Task,
	options *Options,
) contracts.TaskSenderInterface {
	if options == nil {
		options = &Options{
			batchMaxItems:       1000,
			batchTimeout:        50 * time.Millisecond,
			chTasksToConfirmLen: 10000000,
			chBatchesLen:        10000,
		}
	}

	return &taskSender{
		taskManager:        taskManager,
		chTasksToConfirm:   make(chan *domain.Task, options.chTasksToConfirmLen),
		chTasksReadyToSend: chTasksReadyToSend,
		options:            options,
	}
}

type taskSender struct {
	contracts.TaskSenderInterface
	sendByExternalTransport func(task *domain.Task)
	chTasksReadyToSend      <-chan domain.Task
	chTasksToConfirm        chan *domain.Task
	taskManager             contracts.TaskManagerInterface
	options                 *Options
}

func (s *taskSender) SetTransport(sendByExternalTransport func(task *domain.Task)) {
	s.sendByExternalTransport = sendByExternalTransport
}

func (s *taskSender) Send() {
	if s.sendByExternalTransport == nil {
		panic("Transport for sending was not added")
	}
	batchTasksCh := s.generateBatch(s.chTasksToConfirm, s.options.batchMaxItems, s.options.batchTimeout)

	go s.confirmBatch(batchTasksCh)

	for task := range s.chTasksReadyToSend {
		s.sendByExternalTransport(&task)
		s.chTasksToConfirm <- &task
	}
}

func (s *taskSender) confirmBatch(batchTasksCh chan []*domain.Task) {
	for i := 0; i < 5; i++ {
		go func() {
			for batch := range batchTasksCh {
				if err := s.taskManager.ConfirmExecution(batch); err != nil {
					panic(err)
				}
			}
		}()
	}
}

func (s *taskSender) generateBatch(tasks <-chan *domain.Task, maxItems int, maxTimeout time.Duration) chan []*domain.Task {
	batches := make(chan []*domain.Task, s.options.chBatchesLen)

	go func() {
		for {
			var batch []*domain.Task
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
			}
		}
	}()

	return batches
}
