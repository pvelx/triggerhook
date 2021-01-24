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

func (tqh *heapPrioritizedTaskList) Add(task domain.Task) {
	tqh.Lock()
	defer tqh.Unlock()
	item := &item{
		task:     task,
		priority: task.ExecTime,
	}
	heap.Push(&tqh.pq, item)
	tqh.index[task.Id] = &item.index
}

func (tqh *heapPrioritizedTaskList) DeleteIfExist(taskId string) bool {
	tqh.Lock()
	defer tqh.Unlock()
	index, ok := tqh.index[taskId]
	if ok {
		heap.Remove(&tqh.pq, *index)
		delete(tqh.index, taskId)
	}

	return ok
}

func (tqh *heapPrioritizedTaskList) Take() *domain.Task {
	tqh.Lock()
	defer tqh.Unlock()
	if tqh.pq.Len() > 0 {
		task := heap.Pop(&tqh.pq).(*item).task.(domain.Task)
		delete(tqh.index, task.Id)

		return &task
	}
	return nil
}

func (tqh *heapPrioritizedTaskList) Len() int {
	tqh.Lock()
	defer tqh.Unlock()
	return tqh.pq.Len()
}
