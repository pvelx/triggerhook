package task_manager

import (
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/error_service"
	"github.com/pvelx/triggerHook/repository"
	"github.com/pvelx/triggerHook/util"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTaskManager_Delete(t *testing.T) {
	tests := []struct {
		name                        string
		inputErrorRepository        []error
		inputTask                   domain.Task
		expectedError               error
		countCallMethodOfRepository int
		expectedEvents              []string
	}{
		{
			name:                        "main flow - without error",
			inputErrorRepository:        []error{nil},
			expectedError:               nil,
			countCallMethodOfRepository: 1,
			expectedEvents:              []string{},
		},
		{
			name:                        "1 times retryable error",
			inputErrorRepository:        []error{contracts.Deadlock, nil},
			expectedError:               nil,
			countCallMethodOfRepository: 2,
			expectedEvents:              []string{contracts.Deadlock.Error()},
		},
		{
			name:                        "2 times retryable error",
			inputErrorRepository:        []error{contracts.Deadlock, contracts.Deadlock, nil},
			expectedError:               nil,
			countCallMethodOfRepository: 3,
			expectedEvents:              []string{contracts.Deadlock.Error(), contracts.Deadlock.Error()},
		},
		{
			name:                        "3 times  retryable error",
			inputErrorRepository:        []error{contracts.Deadlock, contracts.Deadlock, contracts.Deadlock, nil},
			expectedError:               contracts.TmErrorDeletingTask,
			countCallMethodOfRepository: 3,
			expectedEvents: []string{
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			countCallMethodOfRepository := 0
			r := &repository.RepositoryMock{DeleteMock: func(tasks []domain.Task) (err error) {
				err = test.inputErrorRepository[countCallMethodOfRepository]
				countCallMethodOfRepository++

				return
			}}

			countCallNewOfEventHandler := 0
			eeh := &error_service.ErrorHandlerMock{NewMock: func(level contracts.Level, eventMessage string, extra map[string]interface{}) {
				assert.Equal(t, contracts.LevelError, level, "must be LevelError")
				assert.Equal(t, test.expectedEvents[countCallNewOfEventHandler], eventMessage, "must be LevelError")
				countCallNewOfEventHandler++
			}}

			tm := New(r, eeh, nil)

			result := tm.Delete(util.NewId())

			assert.Equal(t, test.expectedError, result, "error from task manager is not correct")

			assert.Equal(t, test.countCallMethodOfRepository, countCallMethodOfRepository,
				"is not correct call method delete of repository")

			assert.Equal(t, len(test.expectedEvents), countCallNewOfEventHandler,
				"is not correct count of event")
		})
	}
}

func TestTaskManager_Create(t *testing.T) {
	tests := []struct {
		name                        string
		inputErrorRepository        []error
		inputTask                   domain.Task
		expectedError               error
		countCallMethodOfRepository int
		expectedEvents              []string
	}{
		{
			name:                        "main flow - without error",
			inputErrorRepository:        []error{nil},
			expectedError:               nil,
			countCallMethodOfRepository: 1,
			expectedEvents:              []string{},
		},
		{
			name:                        "3 times retryable error",
			inputErrorRepository:        []error{contracts.Deadlock, contracts.Deadlock, contracts.Deadlock, nil},
			expectedError:               contracts.TmErrorCreatingTasks,
			countCallMethodOfRepository: 3,
			expectedEvents: []string{
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			countCallMethodOfRepository := 0
			r := &repository.RepositoryMock{CreateMock: func(task domain.Task, isTaken bool) (err error) {
				err = test.inputErrorRepository[countCallMethodOfRepository]
				countCallMethodOfRepository++

				return
			}}

			countCallNewOfEventHandler := 0
			eeh := &error_service.ErrorHandlerMock{NewMock: func(level contracts.Level, eventMessage string, extra map[string]interface{}) {
				assert.Equal(t, contracts.LevelError, level, "must be LevelError")
				assert.Equal(t, test.expectedEvents[countCallNewOfEventHandler], eventMessage, "must be LevelError")
				countCallNewOfEventHandler++
			}}

			tm := New(r, eeh, nil)

			result := tm.Create(&domain.Task{}, true)

			assert.Equal(t, test.expectedError, result, "error from task manager is not correct")

			assert.Equal(t, test.countCallMethodOfRepository, countCallMethodOfRepository,
				"is not correct call method delete of repository")

			assert.Equal(t, len(test.expectedEvents), countCallNewOfEventHandler,
				"is not correct count of event")
		})
	}
}

func TestTaskManager_ConfirmExecution(t *testing.T) {
	tests := []struct {
		name                        string
		inputErrorRepository        []error
		inputTask                   domain.Task
		expectedError               error
		countCallMethodOfRepository int
		expectedEvents              []string
	}{
		{
			name:                        "main flow - without error",
			inputErrorRepository:        []error{nil},
			expectedError:               nil,
			countCallMethodOfRepository: 1,
			expectedEvents:              []string{},
		},
		{
			name:                        "3 times retryable error",
			inputErrorRepository:        []error{contracts.Deadlock, contracts.Deadlock, contracts.Deadlock, nil},
			expectedError:               contracts.TmErrorConfirmationTasks,
			countCallMethodOfRepository: 3,
			expectedEvents: []string{
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			countCallMethodOfRepository := 0
			r := &repository.RepositoryMock{DeleteMock: func(tasks []domain.Task) (err error) {
				err = test.inputErrorRepository[countCallMethodOfRepository]
				countCallMethodOfRepository++

				return
			}}

			countCallNewOfEventHandler := 0
			eeh := &error_service.ErrorHandlerMock{NewMock: func(level contracts.Level, eventMessage string, extra map[string]interface{}) {
				assert.Equal(t, contracts.LevelError, level, "must be LevelError")
				assert.Equal(t, test.expectedEvents[countCallNewOfEventHandler], eventMessage, "must be LevelError")
				countCallNewOfEventHandler++
			}}

			tm := New(r, eeh, nil)

			result := tm.ConfirmExecution([]domain.Task{{}, {}, {}})

			assert.Equal(t, test.expectedError, result, "error from task manager is not correct")

			assert.Equal(t, test.countCallMethodOfRepository, countCallMethodOfRepository,
				"is not correct call method delete of repository")

			assert.Equal(t, len(test.expectedEvents), countCallNewOfEventHandler,
				"is not correct count of event")
		})
	}
}

func TestTaskManagerMock_GetTasksToComplete(t *testing.T) {
	tests := []struct {
		name             string
		repositoryResult []struct {
			error      error
			collection contracts.CollectionsInterface
		}
		inputTask                   domain.Task
		expectedError               error
		expectedResult              contracts.CollectionsInterface
		countCallMethodOfRepository int
		expectedEvents              []string
	}{
		{
			name: "main flow - without error",
			repositoryResult: []struct {
				error      error
				collection contracts.CollectionsInterface
			}{
				{nil, &repository.Collections{}},
			},
			expectedResult:              &repository.Collections{},
			expectedError:               nil,
			countCallMethodOfRepository: 1,
			expectedEvents:              []string{},
		},
		{
			name: "3 times retryable error",
			repositoryResult: []struct {
				error      error
				collection contracts.CollectionsInterface
			}{
				{contracts.Deadlock, nil},
				{contracts.Deadlock, nil},
				{contracts.Deadlock, nil},
				{nil, &repository.Collections{}},
			},
			expectedError:               contracts.TmErrorGetTasks,
			expectedResult:              nil,
			countCallMethodOfRepository: 3,
			expectedEvents: []string{
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
				contracts.Deadlock.Error(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			countCallMethodOfRepository := 0
			r := &repository.RepositoryMock{FindBySecToExecTimeMock: func(preloadingTimeRange time.Duration) (
				collection contracts.CollectionsInterface,
				err error,
			) {
				err = test.repositoryResult[countCallMethodOfRepository].error
				collection = test.repositoryResult[countCallMethodOfRepository].collection
				countCallMethodOfRepository++

				return
			}}

			countCallNewOfEventHandler := 0
			eeh := &error_service.ErrorHandlerMock{NewMock: func(level contracts.Level, eventMessage string, extra map[string]interface{}) {
				assert.Equal(t, contracts.LevelError, level, "must be LevelError")
				assert.Equal(t, test.expectedEvents[countCallNewOfEventHandler], eventMessage, "must be LevelError")
				countCallNewOfEventHandler++
			}}

			tm := New(r, eeh, nil)

			result, err := tm.GetTasksToComplete(time.Second)

			assert.Equal(t, test.expectedResult, result, "result from task manager is not correct")
			assert.Equal(t, test.expectedError, err, "error from task manager is not correct")

			assert.Equal(t, test.countCallMethodOfRepository, countCallMethodOfRepository,
				"is not correct call method delete of repository")

			assert.Equal(t, len(test.expectedEvents), countCallNewOfEventHandler,
				"is not correct count of event")
		})
	}
}
