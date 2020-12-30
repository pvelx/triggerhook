package repository

import (
	"fmt"
	"github.com/pvelx/triggerHook/domain"
	uuid "github.com/satori/go.uuid"
	"testing"
	"time"
)

//func BenchmarkBunchDelete(b *testing.B) {
//	clear()
//	b.ReportAllocs()
//	b.ResetTimer()
//	b.StopTimer()
//
//	for n := 0; n < b.N; n++ {
//		var tasks []*domain.Task
//		for i := 0; i < 50; i++ {
//			task := &domain.Task{ExecTime: 1}
//			tasks = append(tasks, task)
//			err := repository.Create(task, false)
//			if err != nil {
//				log.Fatal(err, "Error while create")
//			}
//		}
//
//		b.StartTimer()
//		errDelete := repository.DeleteBunch(tasks)
//		if errDelete != nil {
//			log.Fatal(errDelete, "Error while delete")
//		}
//		b.StopTimer()
//	}
//}

//func BenchmarkBunchChangeStatus(b *testing.B) {
//	repo := setUp()
//	b.ReportAllocs()
//	b.ResetTimer()
//	b.StopTimer()
//
//	for n := 0; n < b.N; n++ {
//		var tasks []*domain.Task
//		for i := 0; i < 50; i++ {
//			task := &domain.Task{ExecTime: 1}
//			tasks = append(tasks, task)
//			err := repo.Create(task, false)
//			if err != nil {
//				fmt.Println("err")
//			}
//		}
//
//		b.StartTimer()
//		err := repo.ChangeStatusToCompletedBunch(tasks)
//		b.StopTimer()
//		if err != nil {
//			fmt.Println(err)
//		}
//	}
//}

//func Benchmark4(b *testing.B) {
//	repo := setUp()
//	b.ReportAllocs()
//	b.ResetTimer()
//	b.StopTimer()
//
//	//b.RunParallel(func(pb *testing.PB) {
//	//	for pb.Next() {
//	//		task := &domain.Task{ExecTime: 1}
//	//		err := repo.Create(task, false)
//	//		if err != nil {
//	//			fmt.Println("err")
//	//		}
//	//		b.StartTimer()
//	//		repo.Delete(task)
//	//		b.StopTimer()
//	//	}
//	//})
//
//	for n := 0; n < b.N; n++ {
//
//		task := &domain.Task{ExecTime: 1}
//		err := repo.Create(task, false)
//		if err != nil {
//			fmt.Println("err")
//		}
//		b.StartTimer()
//		repo.Delete(task)
//		b.StopTimer()
//		//fmt.Println(len(tasks))
//	}
//}

func Benchmark3(b *testing.B) {
	clear()
	repository := NewRepository(db, appInstanceId, ErrorHandler{}, &Options{1000})

	b.ReportAllocs()
	b.ResetTimer()
	b.StartTimer()

	for n := 0; n < b.N; n++ {

		task := domain.Task{Id: uuid.NewV4().String(), ExecTime: time.Now().Unix()}
		err := repository.Create(task, false)
		if err != nil {
			fmt.Println(err)
		}
	}
}
