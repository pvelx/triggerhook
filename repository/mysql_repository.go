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
	"sync"
	"time"
)

var NoTasksFound = errors.New("no tasks found")

type Options struct {
	maxCountTasksInCollection int
}

func NewRepository(
	client *sql.DB, appInstanceId string,
	eventErrorHandler contracts.EventErrorHandlerInterface,
	options *Options,
) contracts.RepositoryInterface {
	if options == nil {
		options = &Options{maxCountTasksInCollection: 1000}
	}

	return &mysqlRepository{client, appInstanceId, eventErrorHandler, options}
}

type mysqlRepository struct {
	client            *sql.DB
	appInstanceId     string
	eventErrorHandler contracts.EventErrorHandlerInterface
	options           *Options
}

func (r *mysqlRepository) Create(task domain.Task, isTaken bool) error {
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		log.Fatal(err)
	}

	collectionQueryOp := "!="
	if isTaken {
		collectionQueryOp = "="
	}

	findCollectionQuery := fmt.Sprintf(`SELECT c.id
		FROM collection c
		LEFT JOIN task t on c.id = t.collection_id
		WHERE c.exec_time = ?
			AND c.taken_by_instance %s ?
		GROUP BY c.id
		HAVING count(t.uuid) < ? 
		LIMIT 1`, collectionQueryOp)

	var collectionId int64
	findCollectionErr := tx.QueryRowContext(
		ctx,
		findCollectionQuery,
		task.ExecTime,
		r.appInstanceId,
		r.options.maxCountTasksInCollection,
	).Scan(&collectionId)

	switch {
	case findCollectionErr == sql.ErrNoRows:
		break
	case findCollectionErr != nil:
		if errRollback := tx.Rollback(); errRollback != nil {
			log.Fatal(errRollback)
		}

		return errors.Wrap(findCollectionErr, "find collection error")
	}

	if collectionId == 0 {
		var appInstanceId string
		if isTaken {
			appInstanceId = r.appInstanceId
		}
		creatingCollectionResult, createCollectionsErr := tx.ExecContext(
			ctx,
			"INSERT INTO collection (exec_time, taken_by_instance) VALUE (?, ?)",
			task.ExecTime,
			appInstanceId,
		)
		if createCollectionsErr != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				log.Fatal(errRollback)
			}

			return errors.Wrap(createCollectionsErr, "creating collection is fail")
		}

		collectionId, createCollectionsErr = creatingCollectionResult.LastInsertId()
		if createCollectionsErr != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				log.Fatal(errRollback)
			}

			return errors.Wrap(createCollectionsErr, "creating collection is fail")
		}
	}

	_, createTaskErr := tx.ExecContext(
		ctx,
		"INSERT INTO task (uuid, collection_id) VALUE (?, ?)",
		task.Id,
		collectionId,
	)
	if createTaskErr != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			r.eventErrorHandler.NewEventError(contracts.LevelError, errRollback)
		}

		return errors.Wrap(createTaskErr, "creating task is fail")
	}

	if errCommit := tx.Commit(); errCommit != nil {
		r.eventErrorHandler.NewEventError(contracts.LevelError, errCommit)
	}

	return nil
}

func (r *mysqlRepository) Delete(tasks []domain.Task) error {
	fmt.Println(fmt.Sprintf("DEBUG Repository Delete: start deleting %d tasks", len(tasks)))

	if len(tasks) == 0 {
		return nil
	}

	ctx := context.Background()
	//tx, err := r.client.BeginTx(ctx, &sql.TxOptions{
	//	Isolation: sql.LevelReadUncommitted,
	//})
	//if err != nil {
	//	log.Fatal(err)
	//}

	var deleteTasksArg []interface{}
	for _, task := range tasks {
		deleteTasksArg = append(deleteTasksArg, task.Id)
	}

	params := "?" + strings.Repeat(",?", len(tasks)-1)

	findCollectionsQuery := `SELECT c.id
		FROM collection c WHERE c.exec_time < unix_timestamp()-5
		AND NOT EXISTS(
			SELECT t.uuid FROM task t WHERE t.collection_id = c.id AND t.uuid NOT IN(` + params + `)
		)`

	resultCollectionsId, errExec := r.client.QueryContext(ctx, findCollectionsQuery, deleteTasksArg...)
	if errExec != nil {
		//tx.Rollback()
		return errors.Wrap(errExec, "getting tasks was fail")
	}

	var collectionsIds []interface{}
	for resultCollectionsId.Next() {
		var collectionId int64
		scanCollectionErr := resultCollectionsId.Scan(&collectionId)
		if scanCollectionErr != nil {
			//if errRollback := tx.Rollback(); errRollback != nil {
			//	log.Fatal(errRollback)
			//}

			return errors.Wrap(scanCollectionErr, "scan collection error")
		}
		collectionsIds = append(collectionsIds, collectionId)
	}
	resultCollectionsId.Close()
	fmt.Println(fmt.Sprintf("DEBUG Repository Delete: found %d collections to delete", len(collectionsIds)))

	deleteTaskResult, errDeleteTask := r.client.ExecContext(ctx, "DELETE FROM task WHERE uuid IN ("+params+")", deleteTasksArg...)
	if errDeleteTask != nil {
		//tx.Rollback()
		return errors.Wrap(errDeleteTask, "deleting task was fail")
	}

	deletedRowsCount, errRowsAffected := deleteTaskResult.RowsAffected()
	if errRowsAffected != nil {
		return errors.Wrap(errRowsAffected, "deleting task was fail")
	}
	fmt.Println(fmt.Sprintf("DEBUG Repository Delete: deleted %d tasks", deletedRowsCount))

	if len(collectionsIds) > 0 {
		params2 := "?" + strings.Repeat(",?", len(collectionsIds)-1)
		clearCollectionQuery := `DELETE FROM collection c WHERE id IN(` + params2 + `)`

		clearResult, errClear := r.client.ExecContext(ctx, clearCollectionQuery, collectionsIds...)
		if errClear != nil {
			return errors.Wrap(errClear, "clearing collections was fail")
		}
		affected, errClear := clearResult.RowsAffected()
		if errClear != nil {
			return errors.Wrap(errClear, "clearing collections was fail")
		}
		fmt.Println(fmt.Sprintf("DEBUG Repository Delete: cleaned %d empty collections", affected))
	}

	//if err = tx.Commit(); err != nil {
	//	log.Fatal(err)
	//}

	return nil
}

func (r *mysqlRepository) getTasksByCollection(collectionId int64) (domain.Tasks, error) {
	ctx := context.Background()
	tx, err := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		log.Fatal(err)
	}

	const queryFindBySecToExecTime = `SELECT t.uuid, c.exec_time
		FROM task t
		INNER JOIN collection c on t.collection_id = c.id
		WHERE t.collection_id = ?`

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
	resultTasks.Close()

	fmt.Println(fmt.Sprintf("DEBUG Repository getTasksByCollection: get %d tasks", len(results)))

	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}

	return results, nil
}

func (r *mysqlRepository) FindBySecToExecTime(preloadingTimeRange time.Duration) (contracts.CollectionsInterface, error) {
	toNextExecTime := time.Now().Add(preloadingTimeRange * time.Second).Unix()
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
	resultTasks.Close()
	fmt.Println(fmt.Sprintf("DEBUG Repository FindBySecToExecTime: get %d collections", len(collectionIds)))

	if len(collectionIds) == 0 {
		if err = tx.Commit(); err != nil {
			log.Fatal(err)
		}
		return nil, NoTasksFound
	}

	newQueryLockTasks := fmt.Sprintf(
		"UPDATE collection SET taken_by_instance = ? WHERE id IN(?%s)",
		strings.Repeat(",?", len(collectionIds)-1),
	)

	updateResult, errUpdate := tx.ExecContext(ctx, newQueryLockTasks, args...)
	if errUpdate != nil {
		tx.Rollback()
		return nil, errors.Wrap(errUpdate, "database error")
	}
	rowsAffected, errGetRowsAffected := updateResult.RowsAffected()
	if errGetRowsAffected != nil {
		return nil, errors.Wrap(errUpdate, "database error")
	}

	fmt.Println(fmt.Sprintf("DEBUG Repository FindBySecToExecTime: lock %d collections", rowsAffected))

	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}

	return &Collections{
		mx:          &sync.Mutex{},
		collections: collectionIds,
		r:           r,
	}, nil
}

func (r *mysqlRepository) Up() error {
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
	r           *mysqlRepository
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
