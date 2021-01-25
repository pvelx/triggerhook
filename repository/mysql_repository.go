package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
)

type Options struct {
	/*
		It is approximately count of tasks in collection
	*/
	MaxCountTasksInCollection int

	/*
		0 - disable deleting empty collections
		1 - delete empty collection each times
		n - delete empty collections every n times
	*/
	CleaningFrequency int
}

func New(
	client *sql.DB,
	appInstanceId string,
	eh contracts.EventHandlerInterface,
	options *Options,
) contracts.RepositoryInterface {

	if options == nil {
		options = &Options{}
	}

	if err := mergo.Merge(options, Options{
		MaxCountTasksInCollection: 1000,
		CleaningFrequency:         10,
	}); err != nil {
		panic(err)
	}

	return &mysqlRepository{
		client,
		appInstanceId,
		eh,
		0,
		options,
	}
}

type mysqlRepository struct {
	client            *sql.DB
	appInstanceId     string
	eh                contracts.EventHandlerInterface
	cleanRequestCount int32
	options           *Options
}

func (r *mysqlRepository) Count() (int, error) {
	var count int
	if err := r.client.QueryRow("SELECT count(*) FROM task").Scan(&count); err != nil {
		r.eh.New(contracts.LevelError, err.Error(), nil)

		return 0, contracts.RepoErrorCountingTasks
	}

	return count, nil
}

func (r *mysqlRepository) Create(task domain.Task, isTaken bool) error {
	createTaskQuery := "CALL create_task(?, ?, ?, ?, ?)"
	args := []interface{}{
		r.appInstanceId,
		task.Id,
		task.ExecTime,
		isTaken,
		r.options.MaxCountTasksInCollection,
	}

	if _, err := r.client.Exec(createTaskQuery, args...); err != nil {
		errCreating := contracts.RepoErrorCreatingTask

		if err, ok := err.(*mysql.MySQLError); ok {
			switch {
			case err.Number == mysqlerr.ER_DUP_ENTRY:
				errCreating = contracts.RepoErrorTaskExist
			case err.Number == mysqlerr.ER_LOCK_DEADLOCK:
				errCreating = contracts.RepoErrorDeadlock
			}
		}

		r.eh.New(contracts.LevelError, err.Error(), map[string]interface{}{"task": task})

		return errCreating
	}

	return nil
}

func (r *mysqlRepository) Delete(tasks []domain.Task) (int64, error) {
	if len(tasks) == 0 {
		return 0, nil
	}

	var args []interface{}
	for _, task := range tasks {
		args = append(args, task.Id)
	}

	deletingTaskQuery := fmt.Sprintf("DELETE FROM task WHERE uuid IN (?%s)",
		strings.Repeat(",?", len(tasks)-1))

	result, errDeleting := r.client.Exec(deletingTaskQuery, args...)
	if errDeleting != nil {
		errorReturn := contracts.RepoErrorDeletingTask

		if err, ok := errDeleting.(*mysql.MySQLError); ok {
			switch {
			case err.Number == mysqlerr.ER_LOCK_DEADLOCK:
				errorReturn = contracts.RepoErrorDeadlock
			}
		}

		r.eh.New(contracts.LevelError, errDeleting.Error(), nil)

		return 0, errorReturn
	}

	affected, _ := result.RowsAffected()

	/*
		Cleaning empty collections of tasks.
		It is enough do sometimes.
	*/
	atomic.AddInt32(&r.cleanRequestCount, 1)
	if r.options.CleaningFrequency > 0 &&
		atomic.LoadInt32(&r.cleanRequestCount)%int32(r.options.CleaningFrequency) == 0 {

		if err := r.deleteEmptyCollections(); err != nil {
			r.eh.New(contracts.LevelError, err.Error(), nil)
		}
		atomic.StoreInt32(&r.cleanRequestCount, 0)
	}

	return affected, nil
}

func (r *mysqlRepository) deleteEmptyCollections() error {
	findCollectionsQuery := `SELECT c.id
		FROM collection c WHERE c.exec_time < unix_timestamp()-5
		AND NOT EXISTS(
			SELECT t.uuid FROM task t WHERE t.collection_id = c.id
		)`

	rows, errFinding := r.client.Query(findCollectionsQuery)
	if errFinding != nil {
		return errors.Wrap(errFinding, "deleting empty collections is fail")
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.eh.New(contracts.LevelError, err.Error(), nil)
		}
	}()

	var ids []interface{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return errors.Wrap(err, "scan collection error")
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "scan collection error")
	}

	if len(ids) > 0 {
		deleteCollectionsQuery := fmt.Sprintf("DELETE FROM collection WHERE id IN(?%s)",
			strings.Repeat(",?", len(ids)-1))

		if _, err := r.client.Exec(deleteCollectionsQuery, ids...); err != nil {
			return errors.Wrap(err, "clearing collections was fail")
		}
	}

	return nil
}

func (r *mysqlRepository) getTasksByCollection(collectionId int64) (tasks []domain.Task, error error) {
	queryFindBySecToExecTime := `SELECT t.uuid, c.exec_time
		FROM task t
		INNER JOIN collection c on t.collection_id = c.id
		WHERE t.collection_id = ?`

	rows, errFinding := r.client.Query(queryFindBySecToExecTime, collectionId)
	if errFinding != nil {
		error = contracts.RepoErrorGettingTasks
		r.eh.New(contracts.LevelError, errFinding.Error(), map[string]interface{}{"collection id": collectionId})

		return
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.eh.New(contracts.LevelError, err.Error(), nil)
		}
	}()

	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(&task.Id, &task.ExecTime); err != nil {
			error = contracts.RepoErrorGettingTasks
			r.eh.New(contracts.LevelError, err.Error(), map[string]interface{}{"collection id": collectionId})

			return
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		error = contracts.RepoErrorGettingTasks
		r.eh.New(contracts.LevelError, err.Error(), map[string]interface{}{"collection id": collectionId})

		return
	}

	return
}

func (r *mysqlRepository) FindBySecToExecTime(preloadingTimeRange time.Duration) (collection contracts.CollectionsInterface, error error) {
	toNextExecTime := time.Now().Add(preloadingTimeRange).Unix()
	ctx := context.Background()

	tx, errTx := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if errTx != nil {
		error = contracts.RepoErrorFindingTasks
		r.eh.New(contracts.LevelError, errTx.Error(), nil)

		return
	}

	queryFindBySecToExecTime := `SELECT id 
		FROM collection
		WHERE exec_time <= ? AND taken_by_instance != ?
		ORDER BY exec_time
		FOR UPDATE`

	rows, errFinding := tx.QueryContext(ctx, queryFindBySecToExecTime, toNextExecTime, r.appInstanceId)
	if errFinding != nil {
		error = contracts.RepoErrorFindingTasks

		if err, ok := errFinding.(*mysql.MySQLError); ok {
			switch {
			case err.Number == mysqlerr.ER_LOCK_DEADLOCK:
				error = contracts.RepoErrorDeadlock
			}
		}

		childError := errFinding
		if err := tx.Rollback(); err != nil {
			childError = errors.Wrap(childError, err.Error())
		}
		r.eh.New(contracts.LevelError, childError.Error(), nil)

		return
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.eh.New(contracts.LevelError, err.Error(), nil)
		}
	}()

	var collectionIds []int64
	var args []interface{}
	args = append(args, r.appInstanceId)
	for rows.Next() {
		var collectionId int64
		if err := rows.Scan(&collectionId); err != nil {
			error = contracts.RepoErrorFindingTasks
			r.eh.New(contracts.LevelError, err.Error(), nil)

			return
		}

		args = append(args, collectionId)
		collectionIds = append(collectionIds, collectionId)
	}
	if err := rows.Err(); err != nil {
		childError := err
		if err := tx.Rollback(); err != nil {
			childError = errors.Wrap(childError, err.Error())
		}

		error = contracts.RepoErrorFindingTasks
		r.eh.New(contracts.LevelError, childError.Error(), nil)

		return
	}

	if len(collectionIds) == 0 {
		if err := tx.Commit(); err != nil {
			error = contracts.RepoErrorFindingTasks
			r.eh.New(contracts.LevelError, err.Error(), nil)
			return
		}
		return nil, contracts.RepoErrorNoTasksFound
	}

	newQueryLockTasks := fmt.Sprintf("UPDATE collection SET taken_by_instance = ? WHERE id IN(?%s)",
		strings.Repeat(",?", len(collectionIds)-1))

	if _, err := tx.ExecContext(ctx, newQueryLockTasks, args...); err != nil {
		error = contracts.RepoErrorFindingTasks
		childError := err

		if err, ok := err.(*mysql.MySQLError); ok {
			switch {
			case err.Number == mysqlerr.ER_LOCK_DEADLOCK:
				error = contracts.RepoErrorDeadlock
			}
		}

		if err := tx.Rollback(); err != nil {
			childError = errors.Wrap(childError, err.Error())
		}

		r.eh.New(contracts.LevelError, childError.Error(), nil)

		return
	}

	if err := tx.Commit(); err != nil {
		error = contracts.RepoErrorFindingTasks
		r.eh.New(contracts.LevelError, err.Error(), nil)

		return
	}

	collection = &Collections{
		collections: collectionIds,
		r:           r,
	}

	return
}

func (r *mysqlRepository) Up() (error error) {
	ctx := context.Background()
	tx, errorTx := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if errorTx != nil {
		error = contracts.RepoErrorSchemaSetup
		r.eh.New(contracts.LevelError, errorTx.Error(), nil)

		return
	}

	createCollectionTableQuery := `CREATE TABLE IF NOT EXISTS collection
		(
			id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			exec_time INT NOT NULL,
			taken_by_instance VARCHAR(36) DEFAULT '' NOT NULL,
			INDEX (exec_time)
		)`

	if _, err := tx.ExecContext(ctx, createCollectionTableQuery); err != nil {
		childError := err
		if err := tx.Rollback(); err != nil {
			childError = errors.Wrap(childError, err.Error())
		}

		error = contracts.RepoErrorSchemaSetup
		r.eh.New(contracts.LevelError, childError.Error(), nil)

		return
	}

	createTaskTableQuery := `CREATE TABLE IF NOT EXISTS task
		(
			uuid VARCHAR (36) NOT NULL PRIMARY KEY,
			collection_id BIGINT UNSIGNED NOT NULL ,
			CONSTRAINT task_collection_id_fk FOREIGN KEY (collection_id) REFERENCES collection (id)
		)`

	if _, err := tx.ExecContext(ctx, createTaskTableQuery); err != nil {
		childError := err
		if err := tx.Rollback(); err != nil {
			childError = errors.Wrap(childError, err.Error())
		}

		error = contracts.RepoErrorSchemaSetup
		r.eh.New(contracts.LevelError, childError.Error(), nil)

		return
	}

	createCreateTaskProcedure := `CREATE PROCEDURE create_task(
			param_app_instance VARCHAR(36),
			param_uuid VARCHAR(36),
			param_exec_time INT,
            is_taken BOOL,
            count_task_in_collection INT
        )
		BEGIN
			SET @var_collection_id = 0;
			SET @var_exec_time = param_exec_time;
			SET @var_app_instance = param_app_instance;
			SET @var_count_task_in_collection = count_task_in_collection;
		
			IF is_taken THEN
				SET @app_instance = param_app_instance;
				SET @compare_operator = '=';
			else
				SET @app_instance = '';
				SET @compare_operator = '!=';
			end if;
		
			SET @find_collection_query = CONCAT('SELECT c.id INTO  @var_collection_id
				FROM collection c LEFT JOIN task t on c.id = t.collection_id
				WHERE c.exec_time = ? AND c.taken_by_instance ', @compare_operator, ' ?
				GROUP BY c.id HAVING count(t.uuid) < ? LIMIT 1');
		
			PREPARE stmt FROM @find_collection_query;
			EXECUTE stmt USING @var_exec_time, @var_app_instance, @var_count_task_in_collection;
			DEALLOCATE PREPARE stmt;
		
			IF (@var_collection_id = 0) THEN
				INSERT INTO collection (exec_time, taken_by_instance) VALUE (param_exec_time, @app_instance);
				SET @var_collection_id = LAST_INSERT_ID();
			END IF;
		
			INSERT INTO task (uuid, collection_id) VALUE (param_uuid, @var_collection_id);
		END;`

	if _, errorQuery := tx.ExecContext(ctx, createCreateTaskProcedure); errorQuery != nil {
		mysqlErr, ok := errorQuery.(*mysql.MySQLError)
		if !ok || mysqlErr.Number != mysqlerr.ER_SP_ALREADY_EXISTS {
			error = contracts.RepoErrorSchemaSetup
			childError := errorQuery
			if err := tx.Rollback(); err != nil {
				childError = errors.Wrap(childError, err.Error())
			}

			r.eh.New(contracts.LevelError, childError.Error(), nil)

			return
		}
	}

	if err := tx.Commit(); err != nil {
		error = contracts.RepoErrorFindingTasks
		r.eh.New(contracts.LevelError, err.Error(), nil)

		return
	}

	return
}

type Collections struct {
	sync.Mutex
	collections []int64
	r           *mysqlRepository
}

func (c *Collections) takeCollectionId() (id int64, isEnd bool) {
	c.Lock()
	defer c.Unlock()

	if len(c.collections) == 0 {
		return 0, true
	}
	id = c.collections[0]
	c.collections = c.collections[1:]

	return id, false
}

func (c *Collections) Next() ([]domain.Task, error) {

	id, isEnd := c.takeCollectionId()
	if isEnd {
		return nil, contracts.RepoErrorNoCollections
	}

	tasks, err := c.r.getTasksByCollection(id)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}
