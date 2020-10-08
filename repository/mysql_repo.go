package repository

import (
	"context"
	"fmt"
	"github.com/VladislavPav/trigger-hook/clients/mysql"
	"github.com/VladislavPav/trigger-hook/domain/tasks"
	"github.com/VladislavPav/trigger-hook/utils"
	"log"
	"strconv"
	"strings"
	"time"
)

var MysqlRepo RepositoryInterface = &mysqlRepo{}

const queryCreateTask = "INSERT INTO task (exec_time, status) VALUES (?, ?);"
const queryCreateTaskTaken = "INSERT INTO task (exec_time, taken_by_connection, status) VALUES (?, CONNECTION_ID(), ?);"

const queryFindBySecToExecTime = `SELECT id, exec_time, taken_by_connection, status 
	FROM task
	WHERE status = ? 
		AND exec_time < ? 
		AND (
			taken_by_connection NOT IN (
				SELECT ID FROM INFORMATION_SCHEMA.PROCESSLIST
			) 
			OR taken_by_connection IS NULL
		)`
const queryLockTasks = "UPDATE task SET taken_by_connection = CONNECTION_ID() WHERE id IN(?)"

type RepositoryInterface interface {
	Create(task *tasks.Task, isTaken bool) *utils.ErrorRepo
	FindBySecToExecTime(secToNow int64) (tasks.Tasks, *utils.ErrorRepo)
	ChangeStatusToCompleted(*tasks.Task) *utils.ErrorRepo
}

type mysqlRepo struct{}

func (mysqlRepo) Create(task *tasks.Task, isTaken bool) *utils.ErrorRepo {
	query := queryCreateTask
	if isTaken {
		query = queryCreateTaskTaken
	}
	stmt, err := mysql.Client.Prepare(query)
	if err != nil {
		//logger.Error("Error when trying to prepare save statement", err)
		return utils.NewErrorRepo("database error", err)
	}
	defer stmt.Close()

	insertResult, saveErr := stmt.Exec(task.ExecTime, task.Status)

	if saveErr != nil {
		//logger.Error("Error when trying to save", err)
		return utils.NewErrorRepo("database error", err)
	}

	taskId, errGetId := insertResult.LastInsertId()
	if errGetId != nil {
		//logger.Error("Error when trying to get last id", err)
		return utils.NewErrorRepo("database error", errGetId)
	}
	task.Id = taskId

	return nil
}

func (mysqlRepo) ChangeStatusToCompleted(task *tasks.Task) *utils.ErrorRepo {
	stmt, err := mysql.Client.Prepare("UPDATE task SET status = ? WHERE status = ? AND id = ? taken_by_connection = CONNECTION_ID()")
	if err != nil {
		return utils.NewErrorRepo("database error", err)
	}
	defer stmt.Close()

	_, updateErr := stmt.Exec(tasks.StatusCompleted, tasks.StatusAwaiting, task.Id)

	if updateErr != nil {
		return utils.NewErrorRepo("database error", updateErr)
	}

	return nil
}

func (mysqlRepo) FindBySecToExecTime(secToNow int64) (tasks.Tasks, *utils.ErrorRepo) {
	toNextExecTime := time.Now().Add(time.Duration(secToNow) * time.Second).Unix()
	ctx := context.Background()
	tx, err := mysql.Client.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	resultTasks, errExec := tx.Query(queryFindBySecToExecTime, tasks.StatusAwaiting, toNextExecTime)
	if errExec != nil {
		tx.Rollback()
		fmt.Println("\n", (errExec), "\n ....Transaction rollback!\n")
		return nil, utils.NewErrorRepo("database error", errExec)
	}
	var ids []string
	results := make(tasks.Tasks, 0)
	for resultTasks.Next() {
		var task tasks.Task
		if getErr := resultTasks.Scan(&task.Id, &task.ExecTime, &task.TakenByConnection, &task.Status); getErr != nil {
			return nil, utils.NewErrorRepo("database error", getErr)
		}

		ids = append(ids, strconv.Itoa(int(task.Id)))
		results = append(results, task)
	}
	_, err = tx.Exec(queryLockTasks, strings.Join(ids, ", "))
	if err != nil {
		tx.Rollback()
		fmt.Println("\n", (err), "\n ....Transaction rollback!\n")
		return nil, utils.NewErrorRepo("database error", err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("....Transaction committed\n")
	}

	return results, nil
}
