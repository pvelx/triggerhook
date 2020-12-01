package repository

import (
	"fmt"
	"github.com/VladislavPav/trigger-hook/domain"
	"testing"
)

func Benchmark1(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < 500000; n++ {
		err := MysqlRepo.Create(&domain.Task{Id: int64(n), ExecTime: int64(n)}, false)
		if err != nil {
			fmt.Println("err")
		}
	}
}

func Benchmark2(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		MysqlRepo.FindBySecToExecTime(5)
		//fmt.Println(len(tasks))
	}
}
