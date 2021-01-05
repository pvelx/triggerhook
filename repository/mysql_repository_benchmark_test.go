package repository

import (
	"fmt"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/services"
	"github.com/pvelx/triggerHook/util"
	"log"
	"sync"
	"testing"
	"time"
)

func BenchmarkDelete1000(b *testing.B) {
	benchmarkDelete(1000, b)
}

func BenchmarkDelete500(b *testing.B) {
	benchmarkDelete(500, b)
}

func BenchmarkDelete100(b *testing.B) {
	benchmarkDelete(100, b)
}

func benchmarkDelete(countTaskToDeleteAtOnce int, b *testing.B) {
	b.ResetTimer()
	clear()
	countTaskInCollection := 100
	countCollections := 2000
	mu := sync.Mutex{}

	var collections []collection
	var tasks []task
	taskBunches := make([][]domain.Task, 0, countCollections)
	taskBunch := make([]domain.Task, 0, countTaskInCollection)

	for c := 1; c <= countCollections; c++ {
		collections = append(collections, collection{
			Id:       int64(c),
			ExecTime: time.Now().Unix() - 10,
		})
		for t := 1; t <= countTaskInCollection; t++ {
			tasks = append(tasks, task{
				Id:           util.NewId(),
				CollectionId: int64(c),
			})
		}
	}

	for _, task := range tasks {
		taskBunch = append(taskBunch, domain.Task{Id: task.Id})
		if len(taskBunch) == countTaskToDeleteAtOnce {
			taskBunches = append(taskBunches, taskBunch)
			taskBunch = nil
		}
	}

	upFixtures(collections, tasks)
	repository := NewRepository(db, appInstanceId, &services.ErrorHandlerMock{}, &Options{
		1000,
		10})

	b.SetParallelism(4)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			b.StopTimer()
			mu.Lock()
			if len(taskBunches) == 0 {
				log.Fatal("very small amount of task to delete")
			}
			taskBunch = taskBunches[0]
			taskBunches = taskBunches[1:]
			mu.Unlock()
			b.StartTimer()
			if err := repository.Delete(taskBunch); err != nil {
				log.Fatal(err)
			}
		}
	})
}

func BenchmarkCreate(b *testing.B) {
	clear()
	repository := NewRepository(db, appInstanceId, &services.ErrorHandlerMock{}, &Options{maxCountTasksInCollection: 1000})

	b.ReportAllocs()
	b.ResetTimer()
	b.StartTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := repository.Create(domain.Task{
				Id:       util.NewId(),
				ExecTime: time.Now().Unix(),
			}, false)
			if err != nil {
				fmt.Println(err)
			}
		}
	})
}
