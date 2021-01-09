package prioritized_task_list

import (
	"container/heap"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
)

func NewHeapPrioritizedTaskList(tasks []domain.Task) contracts.PrioritizedTaskListInterface {
	pq := items{}
	var index = make(map[string]int)
	i := 0
	for _, task := range tasks {
		pq = append(pq, &item{
			task:     task,
			priority: task.ExecTime,
			index:    i,
		})
		index[task.Id] = i
		i++
	}
	heap.Init(&pq)

	return &heapPrioritizedTaskList{pq: pq, index: index}
}

type heapPrioritizedTaskList struct {
	contracts.PrioritizedTaskListInterface
	pq    items
	index map[string]int
}

func (tqh *heapPrioritizedTaskList) Add(task domain.Task) {
	tqh.index[task.Id] = tqh.pq.Len()
	heap.Push(&tqh.pq, &item{
		task:     task,
		priority: task.ExecTime,
	})
}

func (tqh *heapPrioritizedTaskList) DeleteIfExist(taskId string) bool {
	index, ok := tqh.index[taskId]
	if ok {
		heap.Remove(&tqh.pq, index)
	}

	return ok
}

func (tqh *heapPrioritizedTaskList) Take() *domain.Task {
	for tqh.pq.Len() > 0 {
		task := heap.Pop(&tqh.pq).(*item).task.(domain.Task)
		return &task
	}
	return nil
}
