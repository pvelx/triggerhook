package prioritized_task_list

import (
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/util"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestShuffleTask(t *testing.T) {
	tasks := getShuffleTasks(int64(1e+5))
	taskHeap := NewHeapPrioritizedTaskList([]domain.Task{})
	for _, task := range tasks {
		taskHeap.Add(task)
	}

	var i int64
	for task := taskHeap.Take(); task != nil; task = taskHeap.Take() {
		assert.Equal(t, i, task.ExecTime)
		i++
	}
}

func getShuffleTasks(countOfTasks int64) []domain.Task {
	rand.Seed(time.Now().UnixNano())
	tasks := make([]domain.Task, 0, countOfTasks)
	for i := int64(0); i < countOfTasks; i++ {
		tasks = append(tasks, domain.Task{Id: util.NewId(), ExecTime: i})
	}
	rand.Shuffle(len(tasks), func(i, j int) { tasks[i], tasks[j] = tasks[j], tasks[i] })

	return tasks
}
