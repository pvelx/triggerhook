package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"log"
	"strings"
	"time"
)

type task struct {
	Id              int64
	ExecTime        int64
	TakenByInstance string
}

func NewRepository(client *sql.DB, appInstanceId string) contracts.RepositoryInterface {
	return &mysqlRepo{client, appInstanceId}
}

type mysqlRepo struct {
	client        *sql.DB
	appInstanceId string
}

func (r *mysqlRepo) Create(task *domain.Task, isTaken bool) error {
	query := "INSERT INTO task (exec_time, taken_by_instance) VALUES (?, ?);"

	stmt, err := r.client.Prepare(query)
	if err != nil {
		return errors.Wrap(err, "database error")
	}
	defer stmt.Close()

	appInstanceId := ""
	if isTaken {
		appInstanceId = r.appInstanceId
	}

	insertResult, saveErr := stmt.Exec(task.ExecTime, appInstanceId)

	if saveErr != nil {
		return errors.Wrap(saveErr, "database error")
	}

	taskId, errGetId := insertResult.LastInsertId()
	if errGetId != nil {
		return errors.Wrap(errGetId, "database error")
	}
	task.Id = taskId

	return nil
}

func (r *mysqlRepo) DeleteBunch(tasks []*domain.Task) error {

	newQueryLockTasks := strings.Replace("DELETE FROM task WHERE id IN (?)", "?", "?"+strings.Repeat(",?", len(tasks)-1), 1)

	stmt, err := r.client.Prepare(newQueryLockTasks)
	if err != nil {
		return errors.Wrap(err, "database error")
	}
	defer stmt.Close()
	var ids []interface{}
	for _, task := range tasks {
		ids = append(ids, task.Id)
	}

	_, updateErr := stmt.Exec(ids...)

	if updateErr != nil {
		return errors.Wrap(updateErr, "database error")
	}

	return nil
}

func (r *mysqlRepo) CountReadyToExec(secToNow int64) (int, error) {
	toNextExecTime := time.Now().Add(time.Duration(secToNow) * time.Second).Unix()
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	const queryCount = `SELECT count(id) FROM task WHERE exec_time <= ? AND taken_by_instance != ?`

	count := 0
	errQuery := tx.QueryRow(queryCount, toNextExecTime, r.appInstanceId).Scan(&count)

	if errQuery != nil {
		tx.Rollback()
		return 0, errors.Wrap(errQuery, "Error getting count of task")
	}

	return count, nil
}

func (r *mysqlRepo) FindBySecToExecTime(secToNow int64, count int) (domain.Tasks, error) {
	toNextExecTime := time.Now().Add(time.Duration(secToNow) * time.Second).Unix()
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	const queryFindBySecToExecTime = `SELECT id, exec_time 
		FROM task
		WHERE exec_time <= ? AND taken_by_instance != ? 
		ORDER BY exec_time
		LIMIT ?
		FOR UPDATE`

	resultTasks, errExec := tx.Query(queryFindBySecToExecTime, toNextExecTime, r.appInstanceId, count)
	if errExec != nil {
		tx.Rollback()
		return nil, errors.Wrap(errExec, "database error")
	}
	ids := make([]interface{}, 0, 1000)
	results := make(domain.Tasks, 0, 1000)
	for resultTasks.Next() {
		var task domain.Task
		if getErr := resultTasks.Scan(&task.Id, &task.ExecTime); getErr != nil {
			return nil, errors.Wrap(getErr, "database error")
		}

		ids = append(ids, int(task.Id))
		results = append(results, task)
	}
	if err = resultTasks.Err(); err != nil {
		log.Fatal(err)
	}

	if len(results) == 0 {
		if err = tx.Commit(); err != nil {
			log.Fatal(err)
		}
		return results, nil
	}

	newQueryLockTasks := fmt.Sprintf("UPDATE task SET taken_by_instance = ? WHERE id IN(?%s)", strings.Repeat(",?", len(ids)-1))

	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, r.appInstanceId)
	args = append(args, ids...)

	_, errUpdate := tx.Exec(newQueryLockTasks, args...)
	if errUpdate != nil {
		tx.Rollback()
		return nil, errors.Wrap(errUpdate, "database error")
	}

	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}

	return results, nil
}

func (r *mysqlRepo) Up() error {
	ctx := context.Background()

	query := `create table IF NOT EXISTS task
		(
			id bigint auto_increment primary key,
			exec_time int default 0 NOT NULL,
			taken_by_instance varchar(36) default '' NOT NULL
		)`

	stmt, err := r.client.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		return err
	}
	return nil
}

//func (r *mysqlRepo) ChangeStatusToCompletedBunch(tasks []*domain.Task) *utils.ErrorRepo {
//	newQuery := "UPDATE task SET status = ? WHERE status = ? AND id IN(?" + strings.Repeat(",?", len(tasks)-1) + ")"
//
//	stmt, err := r.client.Prepare(newQuery)
//	if err != nil {
//		return utils.NewErrorRepo("database error", err)
//	}
//	defer stmt.Close()
//
//	var args []interface{}
//	args = append(args, domain.StatusCompleted)
//	args = append(args, domain.StatusAwaiting)
//	for _, task := range tasks {
//		args = append(args, task.Id)
//	}
//
//	_, updateErr := stmt.Exec(args...)
//	if updateErr != nil {
//		return utils.NewErrorRepo("database error", updateErr)
//	}
//
//	return nil
//}

//func (r *mysqlRepo) ChangeStatusToCompleted(task *domain.Task) *utils.ErrorRepo {
//	stmt, err := r.client.Prepare("UPDATE task SET status = ? WHERE status = ? AND id = ?")
//	if err != nil {
//		return utils.NewErrorRepo("database error", err)
//	}
//	defer stmt.Close()
//
//	_, updateErr := stmt.Exec(domain.StatusCompleted, domain.StatusAwaiting, task.Id)
//
//	if updateErr != nil {
//		return utils.NewErrorRepo("database error", updateErr)
//	}
//
//	return nil
//}

//func (r *mysqlRepo) Delete(task *domain.Task) *utils.ErrorRepo {
//	stmt, err := r.client.Prepare("DELETE FROM task WHERE id = ?")
//	if err != nil {
//		return utils.NewErrorRepo("database error", err)
//	}
//	defer stmt.Close()
//
//	_, updateErr := stmt.Exec(task.Id)
//
//	if updateErr != nil {
//		return utils.NewErrorRepo("database error", updateErr)
//	}
//
//	return nil
//}
