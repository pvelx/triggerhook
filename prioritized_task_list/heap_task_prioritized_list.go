package prioritized_task_list

import (
	"container/heap"
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain"
)

func NewTaskQueueHeap(tasks []domain.Task) contracts.PrioritizedTaskListInterface {
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

	return &heapPrioritizedTaskList{pq: pq}
}

type heapPrioritizedTaskList struct {
	contracts.PrioritizedTaskListInterface
	pq priorityQueue
}

func (tqh *heapPrioritizedTaskList) Add(task *domain.Task) {
	heap.Push(&tqh.pq, &item{
		task:     *task,
		priority: task.ExecTime,
	})
}

func (tqh *heapPrioritizedTaskList) Take() *domain.Task {
	for tqh.pq.Len() > 0 {
		task := heap.Pop(&tqh.pq).(*item).task.(domain.Task)
		return &task
	}
	return nil
}
