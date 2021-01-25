package prioritized_task_list

import (
	"container/heap"
	"sync"

	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

func New(tasks []domain.Task) contracts.PrioritizedTaskListInterface {
	pq := items{}
	var index = make(map[string]*int)
	i := 0
	for _, task := range tasks {
		item := &item{
			task:     task,
			priority: task.ExecTime,
			index:    i,
		}
		pq = append(pq, item)
		index[task.Id] = &item.index
		i++
	}
	heap.Init(&pq)

	return &heapPrioritizedTaskList{pq: pq, index: index}
}

type heapPrioritizedTaskList struct {
	contracts.PrioritizedTaskListInterface
	pq    items
	index map[string]*int
	sync.Mutex
}

func (h *heapPrioritizedTaskList) Add(task domain.Task) {
	h.Lock()
	defer h.Unlock()
	item := &item{
		task:     task,
		priority: task.ExecTime,
	}
	heap.Push(&h.pq, item)
	h.index[task.Id] = &item.index
}

func (h *heapPrioritizedTaskList) DeleteIfExist(taskId string) bool {
	h.Lock()
	defer h.Unlock()
	index, ok := h.index[taskId]
	if ok {
		heap.Remove(&h.pq, *index)
		delete(h.index, taskId)
	}

	return ok
}

func (h *heapPrioritizedTaskList) Take() *domain.Task {
	h.Lock()
	defer h.Unlock()
	if h.pq.Len() > 0 {
		task := heap.Pop(&h.pq).(*item).task.(domain.Task)
		delete(h.index, task.Id)

		return &task
	}
	return nil
}

func (h *heapPrioritizedTaskList) Len() int {
	h.Lock()
	defer h.Unlock()
	return h.pq.Len()
}
