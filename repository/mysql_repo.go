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
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		log.Fatal(err)
	}

	createCollectionQuery := "INSERT INTO collection (exec_time, taken_by_instance)"
	createTaskQuery := "INSERT INTO task (uuid, exec_time, collection_id)"

	var createCollectionArg []interface{}
	var createTasksArg []interface{}
	var execTimes = make(map[int64]bool)
	for i, item := range tasks {
		if _, execTime := execTimes[item.Task.ExecTime]; !execTime {
			collectionQueryOp := "!="
			appInstanceId := ""
			if item.IsTaken {
				collectionQueryOp = "="
				appInstanceId = r.appInstanceId
			}
			createCollectionArg = append(
				createCollectionArg,
				item.Task.ExecTime,
				appInstanceId,
				item.Task.ExecTime,
				item.Task.ExecTime+1,
				r.appInstanceId,
				1000,
			)

			if len(execTimes) != 0 {
				createCollectionQuery = createCollectionQuery + `
				UNION `
			}

			createCollectionQuery = createCollectionQuery + fmt.Sprintf(`
			SELECT ?, ?
			WHERE NOT EXISTS(
					SELECT c.*, count(t.uuid) as task_count
				FROM collection c
				LEFT JOIN task t on c.id = t.collection_id
				WHERE c.exec_time >= ?
					AND c.exec_time < ?
					AND c.taken_by_instance %s ?
				GROUP BY c.id
				HAVING count(c.id) < ?)`, collectionQueryOp)

			execTimes[item.Task.ExecTime] = true
		}

		opTaskQuery := "!="
		if item.IsTaken {
			opTaskQuery = "="
		}

		if item.Task.Id == "" {
			item.Task.Id = uuid.NewV4().String()
		}

		if i != 0 {
			createTaskQuery = createTaskQuery + `
				UNION `
		}
		createTaskQuery = createTaskQuery + fmt.Sprintf(`
			SELECT ?, ?, (
				SELECT id
				FROM collection
				WHERE exec_time = ?
				AND taken_by_instance %s ? LIMIT 1)`, opTaskQuery)

		createTasksArg = append(createTasksArg, item.Task.Id, item.Task.ExecTime, item.Task.ExecTime, r.appInstanceId)
	}
	_, createCollectionsErr := tx.ExecContext(ctx, createCollectionQuery, createCollectionArg...)
	if createCollectionsErr != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			log.Fatal(errRollback)
		}

		return errors.Wrap(createCollectionsErr, "database error")
	}

	_, createTaskErr := tx.ExecContext(ctx, createTaskQuery, createTasksArg...)
	if createTaskErr != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			log.Fatal(errRollback)
		}
		return errors.Wrap(createTaskErr, "database error")
	}

	if errCommit := tx.Commit(); errCommit != nil {
		log.Fatal(errCommit)
	}

	return nil
}

func (r *mysqlRepo) Delete(tasks []*domain.Task) error {
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

	if err = r.ClearEmptyCollection(); err != nil {
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

	if err = tx.Commit(); err != nil {
		log.Fatal(err)
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
	tx, err := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		log.Fatal(err)
	}

	query1 := `create table if not exists collection
		(
			id bigint auto_increment primary key,
			exec_time int not null,
			taken_by_instance varchar(36) default '' not null,
			index (exec_time)
		)`

	_, err1 := tx.ExecContext(ctx, query1)
	if err1 != nil {
		tx.Rollback()
		return err1
	}

	query2 := `create table if not exists task
		(
			uuid varchar(36) not null primary key,
			exec_time int default 0 not null,
			collection_id bigint not null,
			constraint task_collection_id_fk foreign key (collection_id) references collection (id)
		)`

	_, err2 := tx.ExecContext(ctx, query2)
	if err2 != nil {
		tx.Rollback()
		return err2
	}

	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}

	return nil
}

type Collections struct {
	mx          *sync.Mutex
	collections []int64
	r           *mysqlRepo
}

func (c *Collections) takeCollectionId() (id int64, isEnd bool) {
	c.mx.Lock()
	defer func() {
		c.mx.Unlock()
	}()
	if len(c.collections) == 0 {
		return 0, true
	}
	id = c.collections[0]
	c.collections = c.collections[1:]

	return id, false
}

func (c *Collections) Next() (tasks []domain.Task, isEnd bool, err error) {
	err = nil
	var id int64
	id, isEnd = c.takeCollectionId()
	if isEnd {
		return
	}

	tasks, err = c.r.getTasksByCollection(id)
	if err != nil {
		err = errors.New("something wrong")
		return
	}

	return
}
