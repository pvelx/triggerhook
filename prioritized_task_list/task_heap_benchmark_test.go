package prioritized_task_list

import (
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/util"
	"math/rand"
	"testing"
	"time"
)

func BenchmarkAdd(b *testing.B) {
	rand.Seed(time.Now().UnixNano())

	b.ReportAllocs()
	priorityList := New([]domain.Task{})
	for n := 0; n < b.N; n++ {
		execTime := int64(rand.Intn(1000000))
		priorityList.Add(domain.Task{Id: util.NewId(), ExecTime: execTime})
	}
}

func BenchmarkTake(b *testing.B) {
	priorityList := New([]domain.Task{})
	countOfTasks := int64(3e+6)
	for i := int64(0); i < countOfTasks; i++ {
		priorityList.Add(domain.Task{Id: util.NewId(), ExecTime: i})
	}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		priorityList.Take()
	}
}

func BenchmarkBoth(b *testing.B) {
	priorityList := New([]domain.Task{})
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		priorityList.Add(domain.Task{Id: util.NewId(), ExecTime: int64(n)})
		priorityList.Take()
	}
}
