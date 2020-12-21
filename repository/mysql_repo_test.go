package repository

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"sort"
	"sync"
	"testing"
	"time"
)

var (
	db       *sql.DB
	fixtures *testfixtures.Loader

	containerName = "trigger-hook-db"

	dialect  = "mysql"
	user     = "root"
	password = "secret"
	dbName   = "test_db"
	port     = "3307"
	dsn      = "%s:%s@tcp(127.0.0.1:%s)/%s?charset=utf8"

	maxConn  = 25
	idleConn = 25

	appInstanceId = uuid.New().String()

	repository contracts.RepositoryInterface
)

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	var resource *dockertest.Resource

	//Delete old container if one was not deleted in previous test due to fatal error
	if err = pool.RemoveContainerByName(containerName); err != nil {
		log.Fatalf("Does not deleted: %s", err)
	}

	opts := dockertest.RunOptions{
		Name:       containerName,
		Repository: "mysql",
		Tag:        "8.0",
		Env: []string{
			"MYSQL_ROOT_PASSWORD=" + password,
			"MYSQL_DATABASE=" + dbName,
			"MYSQL_PASSWORD=" + password,
		},
		ExposedPorts: []string{"3306"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"3306": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}

	var errRun error
	resource, errRun = pool.RunWithOptions(&opts)
	if errRun != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	dsn = fmt.Sprintf(dsn, user, password, port, dbName)

	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open(dialect, dsn)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	db.SetMaxIdleConns(idleConn)
	db.SetMaxOpenConns(maxConn)

	repository = NewRepository(db, appInstanceId)

	if err := repository.Up(); err != nil {
		log.Fatalf("Up schema is fail: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func loadFixtures(tasks []task) {
	var data = make(map[string]interface{})
	data["tasks"] = tasks

	var err error
	fixtures, err = testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Template(),
		testfixtures.TemplateData(data),
		testfixtures.Dialect("mysql"),
		testfixtures.Files("../test_data/task.yml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := fixtures.Load(); err != nil {
		log.Fatal(err)
	}
}

func TestFindBySecToExecTime(t *testing.T) {
	clear()
	chunkTaskSize := 20
	taskCount := 20

	expectedCountTaskOnIteration := []int{
		chunkTaskSize,
		chunkTaskSize,
		chunkTaskSize,
		chunkTaskSize,
		chunkTaskSize,
		chunkTaskSize,
		chunkTaskSize,
		chunkTaskSize,
		0,
		0,
	}

	loadFixtures(
		NewFixtureTaskBuilder(appInstanceId).
			AddTasksNotTaken(7, taskCount).  //must NOT be get
			AddTasksNotTaken(-6, taskCount). //must be get
			AddTasksNotTaken(-5, taskCount). //must be get
			AddTasksNotTaken(-4, taskCount). //must be get
			AddTasksNotTaken(0, taskCount).  //must be get
			AddTasksNotTaken(4, taskCount).  //must be get
			AddTasksNotTaken(5, taskCount).  //must be get

			AddTasksTakenByCurrentInstance(-10, taskCount). //must NOT be get
			AddTasksTakenByCurrentInstance(0, taskCount).   //must NOT be get
			AddTasksTakenByCurrentInstance(10, taskCount).  //must NOT be get

			AddTasksTakenByBrokenInstance(-10, taskCount). //must be get
			AddTasksTakenByBrokenInstance(0, taskCount).   //must be get
			AddTasksTakenByBrokenInstance(10, taskCount).  //must NOT be get

			GetTasks(),
	)

	var tasksFound = make([]domain.Task, 0, 2000)
	for i := 0; i < len(expectedCountTaskOnIteration); i++ {
		tasks, err := repository.FindBySecToExecTime(5, chunkTaskSize)
		if err != nil {
			log.Fatal(err, "Error while get tasks")
		}

		// TODO check in db

		assert.Equal(
			t,
			expectedCountTaskOnIteration[i],
			len(tasks),
			fmt.Sprintf("Founded count of task for iteration:%d is not correct", i),
		)
		tasksFound = append(tasksFound, tasks...)
	}

	assert.Equal(
		t,
		taskCount*8,
		len(tasksFound),
		"Founded count of task is not correct",
	)
}

// Testing race condition in case parallel access to tasks
func Test_FindBySecToExecTimeRaceCondition(t *testing.T) {
	clear()

	expectedTaskCount := 1000
	chunkTaskSize := 25
	workersCount := 20
	countRequestForWorker := 3

	fixtureTasks := NewFixtureTaskBuilder(appInstanceId).
		AddTasksNotTaken(-4, 500).
		AddTasksTakenByBrokenInstance(-5, 500).
		GetTasks()

	loadFixtures(fixtureTasks)

	workersDone := sync.WaitGroup{}
	workersDone.Add(workersCount)

	startWorkers := make(chan struct{})
	foundTasks := make(chan domain.Task, expectedTaskCount*5)

	for worker := 0; worker < workersCount; worker++ {
		workerNum := worker
		go func() {
			t.Log(fmt.Sprintf("Start worker:%d", workerNum))
			defer workersDone.Done()
			<-startWorkers

			for i := 0; i < countRequestForWorker; i++ {
				tasks, err := repository.FindBySecToExecTime(0, chunkTaskSize)
				if err != nil {
					log.Fatal(err, "Error while get tasks")
				}
				t.Log(fmt.Sprintf(
					"WorkerNum:%d get %d count of tasks. Connections - InUse:%d Idle:%d",
					workerNum,
					len(tasks),
					db.Stats().InUse,
					db.Stats().Idle,
				))

				for _, task := range tasks {
					foundTasks <- task
				}
			}
		}()
	}

	close(startWorkers)

	workersDone.Wait()
	close(foundTasks)

	var taskIdsFound = make([]domain.Task, 0, expectedTaskCount*5)
	for taskFound := range foundTasks {
		taskIdsFound = append(taskIdsFound, taskFound)
	}
	sortTaskById(taskIdsFound)

	for i := 1; i < expectedTaskCount; i++ {
		assert.True(t,
			fixtureTasks[i].Id == taskIdsFound[i].Id &&
				fixtureTasks[i].ExecTime == taskIdsFound[i].ExecTime,
			"Tasks is not equal",
		)
	}
	assert.Len(t, fixtureTasks, expectedTaskCount, "Count of tasks is not equal")
}

//func TestDeleteBunch(t *testing.T) {
//	clear()
//
//	taskCount := 100
//	deleteInOneIteration := 50
//
//	fixtureTasks := NewFixtureTaskBuilder(appInstanceId).
//		AddTasksNotTaken(-100, taskCount).
//		AddTasksNotTaken(100, taskCount).
//		AddTasksTakenByCurrentInstance(-100, taskCount).
//		AddTasksTakenByCurrentInstance(100, taskCount).
//		AddTasksTakenByBrokenInstance(-100, taskCount).
//		AddTasksTakenByBrokenInstance(100, taskCount).
//		GetTasks()
//
//	loadFixtures(fixtureTasks)
//
//	var tasksToDelete = make([]*domain.Task, 0, 2000)
//	for _, taskFixture := range fixtureTasks {
//		tasksToDelete = append(tasksToDelete, &domain.Task{Id: taskFixture.Id})
//	}
//
//	for i := 0; i < len(tasksToDelete); i += deleteInOneIteration {
//		batch := tasksToDelete[i:min(i+deleteInOneIteration, len(tasksToDelete))]
//		err := repository.DeleteBunch(batch)
//		if err != nil {
//			log.Fatal(err, "Error while delete")
//		}
//	}
//
//	// TODO check in db
//}

func TestCreate(t *testing.T) {
	clear()

	tests := []struct {
		name    string
		isTaken bool
	}{
		{"Taken", true},
		{"Not taken", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			for i := 1; i <= 100; i++ {
				errCreate := repository.Create(&domain.Task{ExecTime: time.Now().Unix()}, test.isTaken)
				if errCreate != nil {
					log.Fatal(errCreate, "Error while create")
				}

				// TODO check in db
			}
		})
	}
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func sortTaskById(tasks []domain.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Id < tasks[j].Id
	})
}

func clear() {
	stmt, err := db.Prepare("DELETE FROM task")
	if err != nil {
		log.Fatal(err, "Error clear")
	}
	defer stmt.Close()

	_, updateErr := stmt.Exec()

	if updateErr != nil {
		log.Fatal(updateErr, "Error clear")
	}
}
