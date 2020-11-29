package prioritized_task_list

import (
	"container/heap"
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain"
)

func NewHeapPrioritizedTaskList(tasks []domain.Task) contracts.PrioritizedTaskListInterface {
	pq := items{}
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
	pq items
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

type items []*item

type item struct {
	task     interface{}
	priority int64
	index    int
}

func (items items) Len() int {
	return len(items)
}

func (items items) Less(i, j int) bool {
	return items[i].priority < items[j].priority
}

func (items items) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
	items[i].index = i
	items[j].index = j
}

func (items *items) Push(x interface{}) {
	n := len(*items)
	item := x.(*item)
	item.index = n
	*items = append(*items, item)
}

func (items *items) Pop() interface{} {
	old := *items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*items = old[0 : n-1]
	return item
}
