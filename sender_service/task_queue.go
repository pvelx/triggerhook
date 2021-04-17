package sender_service

import (
	"container/list"
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

type buffer struct {
	In   chan domain.Task
	Out  chan domain.Task
	list *list.List
}

func NewBuffer() *buffer {
	b := &buffer{
		In:   make(chan domain.Task, 1),
		Out:  make(chan domain.Task, 1),
		list: list.New(),
	}
	go b.run()

	return b
}

func (b *buffer) run() {
	for {
		if front := b.list.Front(); front == nil {
			if b.In == nil {
				close(b.Out)
				return
			}
			value, ok := <-b.In
			if !ok {
				close(b.Out)
				return
			}
			b.list.PushBack(value)
		} else {
			select {
			case b.Out <- front.Value.(domain.Task):
				b.list.Remove(front)
			case value, ok := <-b.In:
				if ok {
					b.list.PushBack(value)
				} else {
					b.In = nil
				}
			}
		}
	}
}
