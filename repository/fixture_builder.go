package repository

import (
	"github.com/google/uuid"
	"time"
)

var brokenAppInstanceId string

func init() {
	brokenAppInstanceId = uuid.New().String()
}

func NewFixtureTaskBuilder(currentAppInstanceId string) *fixtureTaskBuilder {
	return &fixtureTaskBuilder{
		currentAppInstanceId: currentAppInstanceId,
	}
}

type fixtureTaskBuilder struct {
	tasks                []task
	currentAppInstanceId string
}

func (b *fixtureTaskBuilder) addTasks(relativeExecTimeSec int64, taskCount int, appInstanceId string) *fixtureTaskBuilder {

	execTime := time.Now().Add(time.Duration(relativeExecTimeSec) * time.Second).Unix()
	var startId = len(b.tasks)
	for i := 1; i <= taskCount; i++ {
		task := task{
			Id:              int64(startId + i),
			ExecTime:        execTime,
			TakenByInstance: appInstanceId,
		}
		if task.Id != 0 {
			b.tasks = append(b.tasks, task)
		}
	}

	return b
}

func (b *fixtureTaskBuilder) AddTasksTakenByCurrentInstance(relativeExecTimeSec int64, taskCount int) *fixtureTaskBuilder {

	b.addTasks(relativeExecTimeSec, taskCount, b.currentAppInstanceId)

	return b
}

func (b *fixtureTaskBuilder) AddTasksTakenByBrokenInstance(relativeExecTimeSec int64, taskCount int) *fixtureTaskBuilder {
	b.addTasks(relativeExecTimeSec, taskCount, brokenAppInstanceId)

	return b
}

func (b *fixtureTaskBuilder) AddTasksNotTaken(relativeExecTimeSec int64, taskCount int) *fixtureTaskBuilder {
	b.addTasks(relativeExecTimeSec, taskCount, "")

	return b
}

func (b *fixtureTaskBuilder) GetTasks() []task {
	return b.tasks
}
