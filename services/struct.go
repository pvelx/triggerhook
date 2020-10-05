package services

import (
	"container/heap"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
)

type Item struct {
	task     tasks.Task
	priority int64
	index    int
}

func NewQueue(initData []tasks.Task) PriorityQueue {
	pq := PriorityQueue{}
	i := 0
	for _, task := range initData {
		pq = append(pq, &Item{
			task:     task,
			priority: task.ExecTime,
			index:    i,
		})
		i++
	}
	heap.Init(&pq)
	return pq
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (p PriorityQueue) Len() int {
	return len(p)
}

func (p PriorityQueue) Less(i, j int) bool {
	return p[i].priority < p[j].priority
}

func (p PriorityQueue) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].index = i
	p[j].index = j
}

func (p *PriorityQueue) Push(x interface{}) {
	n := len(*p)
	item := x.(*Item)
	item.index = n
	*p = append(*p, item)
}

func (p *PriorityQueue) AddTask(task *tasks.Task) {
	item := &Item{
		task:     *task,
		priority: task.ExecTime,
	}
	heap.Push(p, item)
}

func (p *PriorityQueue) Pop() interface{} {
	old := *p
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*p = old[0 : n-1]
	return item
}

func (p *PriorityQueue) GetTask() *tasks.Task {
	for p.Len() > 0 {
		return &heap.Pop(p).(*Item).task
	}
	return nil
}
