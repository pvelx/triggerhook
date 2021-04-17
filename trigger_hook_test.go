package triggerhook

import (
	"context"
	"log"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pvelx/triggerhook/connection"
	"github.com/pvelx/triggerhook/domain"
	"github.com/stretchr/testify/assert"
)

func TestExample(t *testing.T) {
	clear()

	inputData := []struct {
		tasksCount       int32
		relativeExecTime time.Duration
	}{
		{100, -2 * time.Second},
		{200, 0},
		{210, 1 * time.Second},
		{220, 5 * time.Second},
		{230, 7 * time.Second},
		{240, 10 * time.Second},
		{250, 12 * time.Second},
	}

	var expectedAllTasksCount int32
	var maxExecTime time.Duration
	for _, v := range inputData {
		expectedAllTasksCount = expectedAllTasksCount + v.tasksCount
		if maxExecTime < v.relativeExecTime {
			maxExecTime = v.relativeExecTime
		}
	}

	var actualAllTasksCount int32
	triggerHook := Build(Config{
		/*
			If you want to use the access parameters that are not default:

			Connection: connection.Options{
				User:     "user",
				Password: "secret",
				Host:     "127.0.0.1:3306",
				DbName:   "task",
			},
		*/
	})

	go func() {
		for {
			result := triggerHook.Consume()
			now := time.Now().Unix()
			atomic.AddInt32(&actualAllTasksCount, 1)
			assert.Equal(t, now, result.Task().ExecTime, "time exec of the task is not current time")
			result.Confirm()
		}
	}()

	go func() {
		if err := triggerHook.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	for _, current := range inputData {
		for i := int32(0); i < current.tasksCount; i++ {
			execTime := time.Now().Add(current.relativeExecTime).Unix()
			_ = triggerHook.CreateCtx(context.Background(), &domain.Task{ExecTime: execTime})
		}
	}

	time.Sleep(maxExecTime) // it takes time to process the most deferred tasks

	assert.Equal(t, expectedAllTasksCount, atomic.LoadInt32(&actualAllTasksCount), "count tasks is not correct")
}

func clear() {
	conn := connection.New(nil)
	if _, err := conn.Exec("DROP TABLE IF EXISTS task"); err != nil {
		log.Fatal(err)
	}
	if _, err := conn.Exec("DROP TABLE IF EXISTS collection"); err != nil {
		log.Fatal(err)
	}
	conn.Close()
}
