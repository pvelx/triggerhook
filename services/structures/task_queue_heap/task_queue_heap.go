package task_queue_heap

import (
	"container/heap"
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
)

type heapTasksWaitingList struct {
	contracts.TasksWaitingListInterface
	pq priorityQueue
}

// Constructor
func NewTaskQueueHeap(tasks []tasks.Task) contracts.TasksWaitingListInterface {
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

	return &heapTasksWaitingList{pq: pq}
}

func (tqh *heapTasksWaitingList) Add(task *tasks.Task) {
	heap.Push(&tqh.pq, &item{
		task:     *task,
		priority: task.ExecTime,
	})
}

func (tqh *heapTasksWaitingList) Take() *tasks.Task {
	for tqh.pq.Len() > 0 {
		return &heap.Pop(&tqh.pq).(*item).task
	}
	return nil
}
