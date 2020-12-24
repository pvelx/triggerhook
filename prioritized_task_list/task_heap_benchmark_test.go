package prioritized_task_list

import (
	"github.com/pvelx/triggerHook/domain"
	uuid "github.com/satori/go.uuid"
	"math/rand"
	"testing"
	"time"
)

func BenchmarkAdd(b *testing.B) {
	rand.Seed(time.Now().UnixNano())

	b.ReportAllocs()
	priorityList := NewHeapPrioritizedTaskList([]domain.Task{})
	for n := 0; n < b.N; n++ {
		execTime := int64(rand.Intn(1000000))
		priorityList.Add(domain.Task{Id: uuid.NewV4().String(), ExecTime: execTime})
	}
}

func BenchmarkTake(b *testing.B) {
	priorityList := NewHeapPrioritizedTaskList([]domain.Task{})
	countOfTasks := int64(3e+6)
	for i := int64(0); i < countOfTasks; i++ {
		priorityList.Add(domain.Task{Id: uuid.NewV4().String(), ExecTime: i})
	}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		priorityList.Take()
	}
}

func BenchmarkBoth(b *testing.B) {
	priorityList := NewHeapPrioritizedTaskList([]domain.Task{})
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		priorityList.Add(domain.Task{Id: uuid.NewV4().String(), ExecTime: int64(n)})
		priorityList.Take()
	}
}
