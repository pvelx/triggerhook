package prioritized_task_list

import (
	"container/heap"
	"fmt"
	"github.com/VladislavPav/trigger-hook/domain"
	"math/rand"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	ts := data()

	pq := PriorityQueue{}
	i := 0
	for _, task := range ts {
		pq = append(pq, &Item{
			task:     task,
			priority: task.ExecTime,
			index:    i,
		})
		i++
	}
	//fmt.Println("before heapified")
	//for _, f := range pq {
	//	fmt.Println(*f)
	//}

	heap.Init(&pq)

	//fmt.Println("heapified")
	//for _, f := range pq {
	//	//fmt.Println(*f)
	//}

	task := domain.Task{
		Id:       120,
		ExecTime: 3,
	}
	item := &Item{
		task:     task,
		priority: task.ExecTime,
	}
	heap.Push(&pq, item)

	fmt.Println("push")
	for _, f := range pq {
		fmt.Println(*f)
	}
}

func data() []domain.Task {
	//now := time.Now().Unix()
	var idx int64 = 100
	countOfTasks := 10
	var queue []domain.Task
	for i := 0; i < countOfTasks; i++ {
		idx++
		queue = append(queue, domain.Task{Id: idx, ExecTime: int64(i)})
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(queue), func(i, j int) { queue[i], queue[j] = queue[j], queue[i] })

	return queue
}
