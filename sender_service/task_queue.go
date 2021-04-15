package sender_service

import (
	"errors"
	"github.com/pvelx/triggerhook/domain"
	"sync"
)

var QueueIsEmpty = errors.New("queue is empty")

type batchTaskQueue struct {
	sync.Mutex
	queue [][]domain.Task
}

func (t *batchTaskQueue) Push(tasks []domain.Task) {
	t.Lock()
	defer t.Unlock()
	t.queue = append(t.queue, tasks)
}

func (t *batchTaskQueue) Pop() ([]domain.Task, error) {
	t.Lock()
	defer t.Unlock()

	if len(t.queue) > 0 {
		var tasks []domain.Task
		tasks, t.queue = t.queue[0], t.queue[1:]
		return tasks, nil
	}

	return nil, QueueIsEmpty
}
