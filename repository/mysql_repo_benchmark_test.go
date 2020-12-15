package repository

import (
	"github.com/pvelx/triggerHook/domain"
	"log"
	"testing"
)

func BenchmarkBunchDelete(b *testing.B) {
	clear()
	b.ReportAllocs()
	b.ResetTimer()
	b.StopTimer()

	for n := 0; n < b.N; n++ {
		var tasks []*domain.Task
		for i := 0; i < 50; i++ {
			task := &domain.Task{ExecTime: 1}
			tasks = append(tasks, task)
			err := repository.Create(task, false)
			if err != nil {
				log.Fatal(err, "Error while create")
			}
		}

		b.StartTimer()
		errDelete := repository.DeleteBunch(tasks)
		if errDelete != nil {
			log.Fatal(errDelete, "Error while delete")
		}
		b.StopTimer()
	}
}

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

//func Benchmark3(b *testing.B) {
//	repo := setUp()
//	b.ReportAllocs()
//	b.ResetTimer()
//	b.StopTimer()
//
//	//b.RunParallel(func(pb *testing.PB) {
//	//	for pb.Next() {
//	//		task := &domain.Task{ExecTime: 1}
//	//		b.StartTimer()
//	//		err := repo.Create(task, false)
//	//		b.StopTimer()
//	//		if err != nil {
//	//			fmt.Println("err")
//	//		}
//	//
//	//		//repo.ChangeStatusToCompleted(task)
//	//
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
//		repo.ChangeStatusToCompleted(task)
//		b.StopTimer()
//		//fmt.Println(len(tasks))
//	}
//}
