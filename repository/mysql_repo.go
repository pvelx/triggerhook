package repository

import (
	"context"
	"github.com/VladislavPav/trigger-hook/clients"
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain"
	"github.com/VladislavPav/trigger-hook/utils"
	"log"
	"strings"
	"time"
)

var MysqlRepo contracts.RepositoryInterface = &mysqlRepo{}

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
		) LIMIT 20000`
const queryLockTasks = "UPDATE task SET taken_by_connection = CONNECTION_ID() WHERE id IN(?)"

type mysqlRepo struct{}

func (mysqlRepo) Create(task *domain.Task, isTaken bool) *utils.ErrorRepo {
	query := queryCreateTask
	if isTaken {
		query = queryCreateTaskTaken
	}
	stmt, err := clients.Client.Prepare(query)
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

func (mysqlRepo) Delete(taskId string) *error {

	return nil
}

func (mysqlRepo) ChangeStatusToCompleted(task *domain.Task) *utils.ErrorRepo {
	stmt, err := clients.Client.Prepare("UPDATE task SET status = ? WHERE status = ? AND id = ? taken_by_connection = CONNECTION_ID()")
	if err != nil {
		return utils.NewErrorRepo("database error", err)
	}
	defer stmt.Close()

	_, updateErr := stmt.Exec(domain.StatusCompleted, domain.StatusAwaiting, task.Id)

	if updateErr != nil {
		return utils.NewErrorRepo("database error", updateErr)
	}

	return nil
}

func (mysqlRepo) FindBySecToExecTime(secToNow int64) (domain.Tasks, *utils.ErrorRepo) {
	toNextExecTime := time.Now().Add(time.Duration(secToNow) * time.Second).Unix()
	ctx := context.Background()
	tx, err := clients.Client.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	resultTasks, errExec := tx.Query(queryFindBySecToExecTime, domain.StatusAwaiting, toNextExecTime)
	if errExec != nil {
		tx.Rollback()
		//fmt.Println("\n", (errExec), "\n ....Transaction rollback 1!")
		return nil, utils.NewErrorRepo("database error", errExec)
	}
	ids := make([]interface{}, 0, 1000)
	results := make(domain.Tasks, 0, 1000)
	for resultTasks.Next() {
		var task domain.Task
		if getErr := resultTasks.Scan(&task.Id, &task.ExecTime, &task.TakenByConnection, &task.Status); getErr != nil {
			return nil, utils.NewErrorRepo("database error", getErr)
		}

		ids = append(ids, int(task.Id))
		results = append(results, task)
	}

	newQueryLockTasks := strings.Replace(queryLockTasks, "?", "?"+strings.Repeat(",?", len(ids)-1), 1)
	//fmt.Println(newQueryLockTasks)
	_, err = tx.Exec(newQueryLockTasks, ids...)
	if err != nil {
		tx.Rollback()
		//fmt.Println("\n", (err), "\n ....Transaction rollback 2!")
		return nil, utils.NewErrorRepo("database error", err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	} else {
		//fmt.Println("....Transaction committed")
	}

	return results, nil
}
