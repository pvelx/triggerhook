package repository

import (
	"github.com/VladislavPav/trigger-hook/clients/cassandra"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"github.com/VladislavPav/trigger-hook/utils"
	"time"
)

const apiDbLayout = "2006-01-02 15:04:05"

var CassandraRepo RepositoryInterface = &cassandraRepo{}

const queryCreateTask = "INSERT INTO task (formula, next_exec_time, planned_quantity, completed_quantity, deleted_at) VALUES (?, ?, ?, ?, ?)"
const queryFindBySecToExecTime = "SELECT * FROM task WHERE planned_quantity > completed_quantity AND next_exec_time > ?"

type RepositoryInterface interface {
	Create(task tasks.Task) *utils.ErrorRepo
	Delete(task tasks.Task) *utils.ErrorRepo
	Update(task tasks.Task) *utils.ErrorRepo
	FindBySecToExecTime(secToNow int) ([]tasks.Task, *utils.ErrorRepo)
}

type cassandraRepo struct{}

func (cassandraRepo) Create(task tasks.Task) *utils.ErrorRepo {
	task.CompletedQuantity = 0
	task.DeletedAt = ""

	if err := cassandra.GetSession().Query(
		queryCreateTask,
		task.FormulaCalcOfNextTask,
		task.NextExecTime,
		task.PlannedQuantity,
		task.CompletedQuantity,
		task.DeletedAt,
	).Exec(); err != nil {
		return utils.NewErrorRepo("Error when trying to create task", err)
	}
	return nil
}

func (cassandraRepo) Delete(task tasks.Task) *utils.ErrorRepo {
	panic("implement me")
}

func (cassandraRepo) Update(task tasks.Task) *utils.ErrorRepo {
	panic("implement me")
}

func (cassandraRepo) FindBySecToExecTime(secToNow int) ([]tasks.Task, *utils.ErrorRepo) {
	var taskList tasks.Tasks
	m := map[string]interface{}{}
	toNextExecTime := time.Now().Add(-3 * time.Second).Format(apiDbLayout)
	iterable := cassandra.GetSession().Query(queryFindBySecToExecTime, toNextExecTime).Iter()
	for iterable.MapScan(m) {
		taskList = append(taskList, tasks.Task{
			Id:                    m["id"].(int64),
			DeletedAt:             m["deleted_at"].(string),
			NextExecTime:          m["next_exec_time"].(int64),
			CompletedQuantity:     m["completed_quantity"].(int64),
			PlannedQuantity:       m["planned_quantity"].(int64),
			FormulaCalcOfNextTask: m["formula"].(string),
		})
		m = map[string]interface{}{}
	}

	return taskList, nil
}
