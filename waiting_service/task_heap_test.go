package waiting_service

import (
	"math/rand"
	"testing"
	"time"

	"github.com/pvelx/triggerhook/domain"
	"github.com/pvelx/triggerhook/util"
	"github.com/stretchr/testify/assert"
)

func TestShuffleTask(t *testing.T) {
	tasks := getShuffleTasks(int64(1e+5))
	taskHeap := NewPrioritizedTask([]domain.Task{})
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

func TestDeleteTaskFromHeap(t *testing.T) {
	task1 := domain.Task{Id: util.NewId(), ExecTime: 1}
	task2 := domain.Task{Id: util.NewId(), ExecTime: 2}
	task3 := domain.Task{Id: util.NewId(), ExecTime: 3}

	taskHeap := NewPrioritizedTask([]domain.Task{})

	taskHeap.Add(task1)
	taskHeap.Add(task2)
	taskHeap.Add(task3)

	taskHeap.DeleteIfExist(util.NewId())

	task := taskHeap.Take()
	assert.Equal(t, *task, task1)

	task = taskHeap.Take()
	assert.Equal(t, *task, task2)

	taskHeap.DeleteIfExist(task3.Id)

	task = taskHeap.Take()
	assert.Nil(t, task)
}
