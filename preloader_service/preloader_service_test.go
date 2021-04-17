package preloader_service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
	"github.com/pvelx/triggerhook/error_service"
	"github.com/pvelx/triggerhook/monitoring_service"
	"github.com/pvelx/triggerhook/repository"
	"github.com/pvelx/triggerhook/task_manager"
	"github.com/pvelx/triggerhook/util"
	"github.com/stretchr/testify/assert"
)

func TestTaskAdding(t *testing.T) {
	var isTakenActual bool
	taskManagerMock := &task_manager.TaskManagerMock{CreateMock: func(ctx context.Context, task *domain.Task, isTaken bool) error {
		isTakenActual = isTaken
		return nil
	}}

	preloadingService := New(taskManagerMock, nil, &monitoring_service.MonitoringMock{}, nil)

	now := time.Now().Unix()
	tests := []struct {
		task            domain.Task
		isTakenExpected bool
	}{
		{domain.Task{Id: util.NewId(), ExecTime: now - 10}, true},
		{domain.Task{Id: util.NewId(), ExecTime: now - 1}, true},
		{domain.Task{Id: util.NewId(), ExecTime: now + 0}, true},
		{domain.Task{Id: util.NewId(), ExecTime: now + 1}, true},
		{domain.Task{Id: util.NewId(), ExecTime: now + 9}, true},
		{domain.Task{Id: util.NewId(), ExecTime: now + 10}, false},
		{domain.Task{Id: util.NewId(), ExecTime: now + 11}, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			preloadedTask := preloadingService.GetPreloadedChan()
			if err := preloadingService.AddNewTask(context.Background(), &tt.task); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.isTakenExpected, isTakenActual, "The task must be taken")
			if tt.isTakenExpected {
				assert.Equal(t, 1, len(preloadedTask), "Len of channel is wrong")
				assert.Equal(t, tt.task, <-preloadedTask, "The task must be send in channel")
			} else {
				assert.Len(t, preloadedTask, 0, "Len of channel is wrong")
			}
		})
	}
}

func TestMainFlow(t *testing.T) {

	type collectionsType []struct {
		TaskCount int
		Duration  time.Duration
	}

	data := []struct {
		collections collectionsType
	}{
		{
			collections: collectionsType{
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1, Duration: 10 * time.Millisecond},
				{TaskCount: 1, Duration: 10 * time.Millisecond},
				{TaskCount: 1, Duration: 10 * time.Millisecond},
				{TaskCount: 333, Duration: 20 * time.Millisecond},
			},
		},
		{
			collections: collectionsType{
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
				{TaskCount: 1, Duration: 10 * time.Millisecond},
				{TaskCount: 1, Duration: 10 * time.Millisecond},
				{TaskCount: 333, Duration: 20 * time.Millisecond},
			},
		},
		{
			collections: collectionsType{},
		},
		{
			collections: collectionsType{
				{TaskCount: 1000, Duration: 50 * time.Millisecond},
			},
		},
	}

	var globalCurrentFinding int32 = 0

	taskManagerMock := &task_manager.TaskManagerMock{
		GetTasksToCompleteMock: func(ctx context.Context, preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error) {

			currentFinding := atomic.LoadInt32(&globalCurrentFinding)
			if len(data) > int(currentFinding) {
				atomic.AddInt32(&globalCurrentFinding, 1)
				collections := data[currentFinding].collections

				var globalCurrentCollection int32 = 0
				return &repository.CollectionsMock{NextMock: func(ctx context.Context) (tasks []domain.Task, err error) {

					currentCollection := atomic.LoadInt32(&globalCurrentCollection)
					if len(collections) > int(currentCollection) {
						atomic.AddInt32(&globalCurrentCollection, 1)
						collection := collections[currentCollection]
						for i := 0; i < collection.TaskCount; i++ {
							tasks = append(tasks, domain.Task{
								Id:       fmt.Sprintf("%d/%d/%d", currentFinding, currentCollection, i),
								ExecTime: time.Now().Unix(),
							})
						}
						time.Sleep(collection.Duration)

						return tasks, nil
					}

					/*
						Collections of tasks were ended
					*/
					return nil, contracts.RepoErrorNoCollections
				}}, nil
			}

			/*
				Tasks with a suitable time of execute not found
			*/
			return nil, contracts.RepoErrorNoTasksFound
		},
	}

	preloadingService := New(
		taskManagerMock,
		&error_service.ErrorHandlerMock{},
		&monitoring_service.MonitoringMock{},
		nil,
	)

	preloadedTask := preloadingService.GetPreloadedChan()
	go preloadingService.Run()

	var tasks []domain.Task
	go func() {
		for task := range preloadedTask {
			tasks = append(tasks, task)
		}
	}()

	receivedTasks := make(map[string]domain.Task)

	time.Sleep(6 * time.Second)

	for _, item := range data {
		for _, collection := range item.collections {

			if len(tasks) == 0 {
				t.Fatal("tasks was received not enough")
			}

			for i := 0; i < collection.TaskCount; i++ {
				taskActual := tasks[0]
				tasks = tasks[1:]
				if _, exist := receivedTasks[taskActual.Id]; exist {
					t.Fatal(fmt.Sprintf("task '%s' already received", taskActual.Id))
				}
				receivedTasks[taskActual.Id] = taskActual
			}
		}
	}
	assert.Equal(t, 0, len(preloadedTask), "was received extra task")
}
