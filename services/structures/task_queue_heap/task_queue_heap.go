package task_queue_heap

import (
	"container/heap"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"github.com/VladislavPav/trigger-hook/services/structures"
)

type taskQueueHeap struct {
	pq priorityQueue
}

// Constructor
func NewTaskQueueHeap(tasks []tasks.Task) structures.TaskQueueInterface {
	pq := priorityQueue{}
	i := 0
	for _, task := range tasks {
		pq = append(pq, &item{
			task:     task,
			priority: task.ExecTime,
			index:    i,
		})
		i++
	}
	heap.Init(&pq)

	return &taskQueueHeap{pq: pq}
}

func (tqh *taskQueueHeap) Offer(task *tasks.Task) {
	item := &item{
		task:     *task,
		priority: task.ExecTime,
	}
	heap.Push(&tqh.pq, item)
}

func (tqh *taskQueueHeap) Poll() *tasks.Task {
	for tqh.pq.Len() > 0 {
		return &heap.Pop(&tqh.pq).(*item).task
	}
	return nil
}

type priorityQueue []*item

type item struct {
	task     tasks.Task
	priority int64
	index    int
}

func (p priorityQueue) Len() int {
	return len(p)
}

func (p priorityQueue) Less(i, j int) bool {
	return p[i].priority < p[j].priority
}

func (p priorityQueue) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].index = i
	p[j].index = j
}

func (p *priorityQueue) Push(x interface{}) {
	n := len(*p)
	item := x.(*item)
	item.index = n
	*p = append(*p, item)
}

func (p *priorityQueue) Pop() interface{} {
	old := *p
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*p = old[0 : n-1]
	return item
}
