package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/satori/go.uuid"
	"log"
	"strings"
	"sync"
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

func (r *mysqlRepo) Create(tasks []contracts.TaskToCreate) error {

	createCollectionQuery := "INSERT INTO collection (exec_time, taken_by_instance)"
	q := "INSERT INTO task (uuid, exec_time, collection_id)"

	var createCollectionArg []interface{}
	var createTasksArg []interface{}
	var execTimes = make(map[int64]bool)
	for i, item := range tasks {
		op := "!="
		if item.IsTaken {
			op = "="
		}

		if _, execTime := execTimes[item.Task.ExecTime]; !execTime {
			execTimes[item.Task.ExecTime] = true

			appInstanceId := ""
			if item.IsTaken {
				appInstanceId = r.appInstanceId
			}
			createCollectionArg = append(
				createCollectionArg,
				item.Task.ExecTime,
				appInstanceId,
				item.Task.ExecTime,
				item.Task.ExecTime+1,
				appInstanceId,
				1000,
			)

			createCollectionQuery = createCollectionQuery + fmt.Sprintf(`
			SELECT ?, ?
			WHERE NOT EXISTS(
					SELECT c.*, count(t.id) as task_count
				FROM collection c
				LEFT JOIN task t on c.id = t.collection_id
				WHERE c.exec_time >= ?
					AND c.exec_time < ?
					AND c.taken_by_instance %s ?
				GROUP BY c.id
				HAVING count(c.id) < ?)`, op)

			if len(tasks)-1 > i {
				createCollectionQuery = createCollectionQuery + `
				UNION `
			}
		}

		if item.Task.Id == "" {
			item.Task.Id = uuid.NewV4().String()
		}

		q = q + fmt.Sprintf(`
			SELECT ?, ?, (
				SELECT id
				FROM collection
				WHERE exec_time = ?
				AND taken_by_instance %s ?)`, op)

		if len(tasks)-1 > i {
			q = q + `
				UNION `
		}

		createTasksArg = append(createTasksArg, item.Task.Id, item.Task.ExecTime, item.Task.ExecTime, r.appInstanceId)
	}

	stmt2, err2 := r.client.Prepare(q)
	if err2 != nil {
		return errors.Wrap(err2, "database error")
	}

	stmt, err := r.client.Prepare(q)
	if err != nil {
		return errors.Wrap(err, "database error")
	}
	defer stmt.Close()
	defer stmt2.Close()

	_, saveCollectionsErr := stmt2.Exec(createCollectionArg)
	if saveCollectionsErr != nil {
		return errors.Wrap(saveCollectionsErr, "database error")
	}

	_, saveErr := stmt.Exec(createTasksArg)
	if saveErr != nil {
		return errors.Wrap(saveErr, "database error")
	}

	return nil
}

func (r *mysqlRepo) ConfirmExecution(tasks []*domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		log.Fatal(err)
	}

	var deleteTasksArg []interface{}
	for _, task := range tasks {
		deleteTasksArg = append(deleteTasksArg, task.Id)
	}

	deleteTasksQuery := strings.Replace(
		"DELETE FROM task WHERE uuid IN (?)",
		"?",
		"?"+strings.Repeat(",?", len(tasks)-1),
		1,
	)

	_, errDeleteTask := tx.ExecContext(ctx, deleteTasksQuery, deleteTasksArg...)
	if errDeleteTask != nil {
		tx.Rollback()
		return errors.Wrap(errDeleteTask, "deleting task was fail")
	}

	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func (r *mysqlRepo) ClearEmptyCollection() error {
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		log.Fatal(err)
	}

	clearCollectionQuery := `DELETE FROM collection c
		WHERE exec_time < unix_timestamp() - 5
		AND NOT EXISTS(SELECT t.uuid FROM task t WHERE t.collection_id = c.id)`

	_, errClear := tx.ExecContext(ctx, clearCollectionQuery)
	if errClear != nil {
		return errors.Wrap(errClear, "clearing collections was fail")
	}

	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func (r *mysqlRepo) getTasksByCollection(collectionId int64) (domain.Tasks, error) {
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		log.Fatal(err)
	}

	const queryFindBySecToExecTime = `SELECT uuid, exec_time
		FROM task
		WHERE collection_id = ?
		ORDER BY exec_time`

	resultTasks, errExec := tx.QueryContext(ctx, queryFindBySecToExecTime, collectionId)
	if errExec != nil {
		tx.Rollback()
		return nil, errors.Wrap(errExec, "getting tasks was fail")
	}

	results := make(domain.Tasks, 0, 1000)
	for resultTasks.Next() {
		var task domain.Task
		if getErr := resultTasks.Scan(&task.Id, &task.ExecTime); getErr != nil {
			return nil, errors.Wrap(getErr, "getting tasks was fail")
		}
		results = append(results, task)
	}

	return results, nil
}

func (r *mysqlRepo) FindBySecToExecTime(secToNow int64) (contracts.CollectionsInterface, error) {
	toNextExecTime := time.Now().Add(time.Duration(secToNow) * time.Second).Unix()
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		log.Fatal(err)
	}

	const queryFindBySecToExecTime = `SELECT id 
		FROM collection
		WHERE exec_time <= ? AND taken_by_instance != ?
		ORDER BY exec_time
		FOR UPDATE`

	resultTasks, errExec := tx.QueryContext(ctx, queryFindBySecToExecTime, toNextExecTime, r.appInstanceId)
	if errExec != nil {
		tx.Rollback()
		//fmt.Println(errExec)
		return nil, errors.Wrap(errExec, "database error")
	}

	args := make([]interface{}, 0, 1000)
	args = append(args, r.appInstanceId)
	collectionIds := make([]int64, 0, 1000)
	for resultTasks.Next() {
		var collectionId int64
		if getErr := resultTasks.Scan(&collectionId); getErr != nil {
			return nil, errors.Wrap(getErr, "database error")
		}

		args = append(args, collectionId)
		collectionIds = append(collectionIds, collectionId)
	}
	if err = resultTasks.Err(); err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	if len(collectionIds) == 0 {
		if err = tx.Commit(); err != nil {
			log.Fatal(err)
		}
		return nil, errors.New("empty")
	}

	newQueryLockTasks := fmt.Sprintf(
		"UPDATE collection SET taken_by_instance = ? WHERE id IN(?%s)",
		strings.Repeat(",?", len(collectionIds)-1),
	)

	_, errUpdate := tx.ExecContext(ctx, newQueryLockTasks, args...)
	if errUpdate != nil {
		tx.Rollback()
		return nil, errors.Wrap(errUpdate, "database error")
	}

	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}

	return &Collections{
		mx:          &sync.Mutex{},
		collections: collectionIds,
		r:           r,
	}, nil
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

type Collections struct {
	mx          *sync.Mutex
	collections []int64
	r           *mysqlRepo
}

func (c *Collections) takeCollectionId() (int64, bool) {
	c.mx.Lock()
	defer func() {
		c.mx.Unlock()
	}()
	if len(c.collections) == 0 {
		return 0, false
	}
	collectionId := c.collections[0]
	c.collections = c.collections[1:]

	return collectionId, true
}

func (c *Collections) Next() ([]domain.Task, error) {
	collectionId, ok := c.takeCollectionId()
	if !ok {
		return []domain.Task{}, nil
	}

	tasks, err := c.r.getTasksByCollection(collectionId)
	if err != nil {
		return nil, errors.New("something wrong")
	}
	return tasks, nil
}
