package triggerhook

import (
	"context"
	"log"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pvelx/triggerhook/connection"
	"github.com/pvelx/triggerhook/domain"
	"github.com/stretchr/testify/assert"
)

func TestExample(t *testing.T) {
	clear()
	actualAllTasksCount := 0
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
			actualAllTasksCount++
			assert.Equal(t, now, result.Task().ExecTime, "time exec of the task is not current time")
			result.Confirm()
		}
	}()

	go func() {
		if err := triggerHook.Run(); err != nil {
			log.Fatal(err)
		}
	}()
	//it takes time for run trigger hook
	time.Sleep(time.Second)

	inputData := []struct {
		tasksCount       int
		relativeExecTime int64
	}{
		{100, -2},
		{200, 0},
		{210, 1},
		{220, 5},
		{230, 7},
		{240, 10},
		{250, 12},
	}
	expectedAllTasksCount := 0

	for _, current := range inputData {
		expectedAllTasksCount = expectedAllTasksCount + current.tasksCount
		current := current
		go func() {
			for i := 0; i < current.tasksCount; i++ {
				execTime := time.Now().Add(time.Duration(current.relativeExecTime) * time.Second).Unix()
				if err := triggerHook.CreateCtx(context.Background(), &domain.Task{
					ExecTime: execTime,
				}); err != nil {
					t.Fatal(err)
				}
			}
		}()
	}

	// it takes time to process the most deferred tasks
	time.Sleep(13 * time.Second)

	assert.Equal(t, expectedAllTasksCount, actualAllTasksCount, "count tasks is not correct")
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
