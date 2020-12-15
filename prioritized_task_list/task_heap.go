package prioritized_task_list

import (
	"container/heap"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
)

func NewHeapPrioritizedTaskList(tasks []domain.Task) contracts.PrioritizedTaskListInterface {
	pq := items{}
	var index = make(map[int64]int)
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
	index map[int64]int
}

func (tqh *heapPrioritizedTaskList) Add(task *domain.Task) {
	tqh.index[task.Id] = tqh.pq.Len()
	heap.Push(&tqh.pq, &item{
		task:     *task,
		priority: task.ExecTime,
	})
}

func (tqh *heapPrioritizedTaskList) DeleteIfExist(taskId int64) bool {
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
