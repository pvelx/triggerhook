package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Options struct {
	/*
		It is approximately count of tasks in collection
	*/
	maxCountTasksInCollection int

	/*
		0 - disable deleting empty collections
		1 - delete empty collection each times
		n - delete empty collections every n times
	*/
	cleaningFrequency int32
}

func NewRepository(
	client *sql.DB, appInstanceId string,
	eer contracts.EventErrorHandlerInterface,
	options *Options,
) contracts.RepositoryInterface {
	if options == nil {
		/*
			Default options
		*/
		options = &Options{
			maxCountTasksInCollection: 1000,
			cleaningFrequency:         10,
		}
	}

	return &mysqlRepository{
		client,
		appInstanceId,
		eer,
		0,
		options,
	}
}

type mysqlRepository struct {
	client            *sql.DB
	appInstanceId     string
	eer               contracts.EventErrorHandlerInterface
	cleanRequestCount int32
	options           *Options
}

func (r *mysqlRepository) Create(task domain.Task, isTaken bool) error {
	createTaskQuery := "CALL create_task(?, ?, ?, ?, ?)"
	args := []interface{}{
		r.appInstanceId,
		task.Id,
		task.ExecTime,
		isTaken,
		r.options.maxCountTasksInCollection,
	}

	if _, err := r.client.Exec(createTaskQuery, args...); err != nil {
		errCreating := contracts.FailCreatingTask

		if err, ok := err.(*mysql.MySQLError); ok {
			switch {
			case err.Number == mysqlerr.ER_DUP_ENTRY:
				errCreating = contracts.TaskExist
			case err.Number == mysqlerr.ER_LOCK_DEADLOCK:
				errCreating = contracts.Deadlock
			}
		}

		r.eer.New(contracts.LevelError, err.Error(), map[string]interface{}{"task": task})

		return errCreating
	}

	return nil
}

func (r *mysqlRepository) Delete(tasks []domain.Task) (error error) {
	//fmt.Println(fmt.Sprintf("DEBUG Repository Delete: start deleting %d tasks", len(tasks)))
	if len(tasks) == 0 {
		return nil
	}

	var args []interface{}
	for _, task := range tasks {
		args = append(args, task.Id)
	}

	deletingTaskQuery := fmt.Sprintf("DELETE FROM task WHERE uuid IN (?%s)",
		strings.Repeat(",?", len(tasks)-1))

	if _, err := r.client.Exec(deletingTaskQuery, args...); err != nil {
		error = contracts.FailDeletingTask

		if err, ok := err.(*mysql.MySQLError); ok {
			switch {
			case err.Number == mysqlerr.ER_LOCK_DEADLOCK:
				error = contracts.Deadlock
			}
		}

		r.eer.New(contracts.LevelError, err.Error(), nil)

		return
	}

	/*
		Cleaning empty collections of tasks.
		It is enough do sometimes.
	*/
	atomic.AddInt32(&r.cleanRequestCount, 1)
	if r.options.cleaningFrequency > 0 && atomic.LoadInt32(&r.cleanRequestCount)%r.options.cleaningFrequency == 0 {
		if err := r.deleteEmptyCollections(); err != nil {
			r.eer.New(contracts.LevelError, err.Error(), nil)
		}
		atomic.StoreInt32(&r.cleanRequestCount, 0)
	}

	return nil
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
			r.eer.New(contracts.LevelError, err.Error(), nil)
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

	//fmt.Println(fmt.Sprintf("DEBUG Repository deleteEmptyCollections: found %d collections to delete", len(collectionsIds)))

	if len(ids) > 0 {
		deleteCollectionsQuery := fmt.Sprintf("DELETE FROM collection c WHERE id IN(?%s)",
			strings.Repeat(",?", len(ids)-1))

		if _, err := r.client.Exec(deleteCollectionsQuery, ids...); err != nil {
			return errors.Wrap(err, "clearing collections was fail")
		}

		//fmt.Println(fmt.Sprintf("DEBUG Repository deleteEmptyCollections: cleaned %d empty collections", affected))
	}

	return nil
}

func (r *mysqlRepository) getTasksByCollection(collectionId int64) (tasks domain.Tasks, error error) {
	queryFindBySecToExecTime := `SELECT t.uuid, c.exec_time
		FROM task t
		INNER JOIN collection c on t.collection_id = c.id
		WHERE t.collection_id = ?`

	rows, errFinding := r.client.Query(queryFindBySecToExecTime, collectionId)
	if errFinding != nil {
		error = contracts.FailGettingTasks
		r.eer.New(contracts.LevelError, errFinding.Error(), map[string]interface{}{"collection id": collectionId})

		return
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.eer.New(contracts.LevelError, err.Error(), nil)
		}
	}()

	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(&task.Id, &task.ExecTime); err != nil {
			error = contracts.FailGettingTasks
			r.eer.New(contracts.LevelError, err.Error(), map[string]interface{}{"collection id": collectionId})

			return
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		error = contracts.FailGettingTasks
		r.eer.New(contracts.LevelError, err.Error(), map[string]interface{}{"collection id": collectionId})

		return
	}

	//fmt.Println(fmt.Sprintf("DEBUG Repository getTasksByCollection: get %d tasks of collection %d", len(results), collectionId))

	return
}

func (r *mysqlRepository) FindBySecToExecTime(preloadingTimeRange time.Duration) (collection contracts.CollectionsInterface, error error) {
	toNextExecTime := time.Now().Add(preloadingTimeRange).Unix()
	ctx := context.Background()
	//fmt.Println(fmt.Sprintf("DEBUG Repository FindBySecToExecTime: begin tx collection"))
	tx, errTx := r.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if errTx != nil {
		error = contracts.FailFindingTasks
		r.eer.New(contracts.LevelError, errTx.Error(), nil)

		return
	}

	queryFindBySecToExecTime := `SELECT id 
		FROM collection
		WHERE exec_time <= ? AND taken_by_instance != ?
		ORDER BY exec_time
		FOR UPDATE`

	rows, errFinding := tx.QueryContext(ctx, queryFindBySecToExecTime, toNextExecTime, r.appInstanceId)
	if errFinding != nil {
		error = contracts.FailFindingTasks

		if err, ok := errFinding.(*mysql.MySQLError); ok {
			switch {
			case err.Number == mysqlerr.ER_LOCK_DEADLOCK:
				error = contracts.Deadlock
			}
		}

		childError := errFinding
		if err := tx.Rollback(); err != nil {
			childError = errors.Wrap(childError, err.Error())
		}
		r.eer.New(contracts.LevelError, childError.Error(), nil)

		return
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.eer.New(contracts.LevelError, err.Error(), nil)
		}
	}()

	var collectionIds []int64
	var args []interface{}
	args = append(args, r.appInstanceId)
	for rows.Next() {
		var collectionId int64
		if err := rows.Scan(&collectionId); err != nil {
			error = contracts.FailFindingTasks
			r.eer.New(contracts.LevelError, err.Error(), nil)

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

		error = contracts.FailFindingTasks
		r.eer.New(contracts.LevelError, childError.Error(), nil)

		return
	}

	//fmt.Println(fmt.Sprintf("DEBUG Repository FindBySecToExecTime: get %d collections", len(collectionIds)))

	if len(collectionIds) == 0 {
		//fmt.Println(fmt.Sprintf("DEBUG Repository FindBySecToExecTime: end tx collection"))
		if err := tx.Commit(); err != nil {
			error = contracts.FailFindingTasks
			r.eer.New(contracts.LevelError, err.Error(), nil)
			return
		}
		return nil, contracts.NoTasksFound
	}

	newQueryLockTasks := fmt.Sprintf("UPDATE collection SET taken_by_instance = ? WHERE id IN(?%s)",
		strings.Repeat(",?", len(collectionIds)-1))

	if _, err := tx.ExecContext(ctx, newQueryLockTasks, args...); err != nil {
		error = contracts.FailFindingTasks
		childError := err

		if err, ok := err.(*mysql.MySQLError); ok {
			switch {
			case err.Number == mysqlerr.ER_LOCK_DEADLOCK:
				error = contracts.Deadlock
			}
		}

		if err := tx.Rollback(); err != nil {
			childError = errors.Wrap(childError, err.Error())
		}

		r.eer.New(contracts.LevelError, childError.Error(), nil)

		return
	}

	//fmt.Println(fmt.Sprintf("DEBUG Repository FindBySecToExecTime: end tx collection"))
	if err := tx.Commit(); err != nil {
		error = contracts.FailFindingTasks
		r.eer.New(contracts.LevelError, err.Error(), nil)

		return
	}

	collection = &Collections{
		mu:          &sync.Mutex{},
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
		error = contracts.FailSchemaSetup
		r.eer.New(contracts.LevelError, errorTx.Error(), nil)

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

		error = contracts.FailSchemaSetup
		r.eer.New(contracts.LevelError, childError.Error(), nil)

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

		error = contracts.FailSchemaSetup
		r.eer.New(contracts.LevelError, childError.Error(), nil)

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
			error = contracts.FailSchemaSetup
			childError := errorQuery
			if err := tx.Rollback(); err != nil {
				childError = errors.Wrap(childError, err.Error())
			}

			r.eer.New(contracts.LevelError, childError.Error(), nil)

			return
		}
	}

	if err := tx.Commit(); err != nil {
		error = contracts.FailFindingTasks
		r.eer.New(contracts.LevelError, err.Error(), nil)

		return
	}

	return
}

type Collections struct {
	mu          *sync.Mutex
	collections []int64
	r           *mysqlRepository
}

func (c *Collections) takeCollectionId() (id int64, isEnd bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.collections) == 0 {
		return 0, true
	}
	id = c.collections[0]
	c.collections = c.collections[1:]

	return id, false
}

func (c *Collections) Next() (tasks []domain.Task, isEnd bool, error error) {
	var id int64
	id, isEnd = c.takeCollectionId()
	if isEnd {
		return
	}

	tasks, error = c.r.getTasksByCollection(id)

	return
}
