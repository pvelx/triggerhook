package prioritized_task_list

import (
	"fmt"
	"github.com/VladislavPav/trigger-hook/domain"
	"math/rand"
	"testing"
	"time"
)

func TestCommon(t *testing.T) {
	ts := data()
	tqh := NewTaskQueueHeap(ts)

	task := domain.Task{
		Id:       120,
		ExecTime: 3,
	}
	tqh.Offer(&task)

	for task := tqh.Poll(); task != nil; task = tqh.Poll() {
		fmt.Println(task)
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
