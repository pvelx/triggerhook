package repository

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
)

var (
	db *sql.DB

	containerName = "trigger-hook-db"

	dialect  = "mysql"
	user     = "root"
	password = "secret"
	dbName   = "test_db"
	port     = "3307"
	dsn      = "%s:%s@tcp(127.0.0.1:%s)/%s?charset=utf8"

	maxConn  = 25
	idleConn = 25

	appInstanceId = uuid.NewV4().String()

	repository contracts.RepositoryInterface
)

type ErrorHandler struct {
	contracts.EventErrorHandlerInterface
}

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

	repository = NewRepository(db, appInstanceId, ErrorHandler{})

	if err := repository.Up(); err != nil {
		log.Fatalf("Up schema is fail: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func loadFixtures(testDir string) {
	/*
		collection fixture
	*/
	file, err := os.Open(fmt.Sprintf("../test_data/%s/collection.csv", testDir))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 3
	reader.Comment = '#'

	insertCollection := "INSERT INTO collection (id, exec_time, taken_by_instance) VALUES "
	var insertCollectionArgs []interface{}
	now := time.Now().Unix()

	for {
		record, e := reader.Read()
		if e != nil {
			fmt.Println(e)
			break
		}
		relativeExecTime, _ := strconv.ParseInt(record[1], 10, 64)

		if record[2] == "current" {
			record[2] = appInstanceId
		}

		insertCollection = insertCollection + "(?, ?, ?),"
		insertCollectionArgs = append(insertCollectionArgs, record[0], relativeExecTime+now, record[2])
	}
	insertCollection = insertCollection[:len(insertCollection)-len(",")]

	_, err = db.Exec(insertCollection, insertCollectionArgs...)
	if err != nil {
		panic(err)
	}

	/*
		task fixture
	*/
	taskFile, err := os.Open(fmt.Sprintf("../test_data/%s/task.csv", testDir))
	if err != nil {
		panic(err)
	}
	defer taskFile.Close()

	readerTaskFile := csv.NewReader(taskFile)
	readerTaskFile.FieldsPerRecord = 2

	insertTask := "INSERT INTO task (uuid, collection_id) VALUES "
	var values string
	var insertTaskArgs []interface{}

	for {
		recordTask, e := readerTaskFile.Read()
		if e != nil {
			fmt.Println(e)
			break
		}

		values = values + "(?, ?),"
		insertTaskArgs = append(insertTaskArgs, recordTask[0], recordTask[1])

		if len(insertTaskArgs) > 1000 {
			values = values[:len(values)-len(",")]

			_, err = db.Exec(insertTask+values, insertTaskArgs...)
			if err != nil {
				panic(err)
			}

			values = ""
			insertTaskArgs = nil
		}
	}
}

func TestFindBySecToExecTime(t *testing.T) {
	clear()
	loadFixtures("data_2")

	expectedCountTaskOnIteration := []int{
		33, 981, 894, 128, 212, 174, 90, 148, 167, 108, 26, 966, 967, 0,
		835, 140, 538, 127, 209, 356, 605, 354, 591, 0, 0, 0, 0, 0,
	}
	expectedCollectionCount := len(expectedCountTaskOnIteration)

	var countAllTask int
	for _, count := range expectedCountTaskOnIteration {
		countAllTask = countAllTask + count
	}

	collections, err := repository.FindBySecToExecTime(5)
	if err != nil {
		log.Fatal(err, "Error while get tasks")
	}

	allTasks := make(map[string]domain.Task)
	actualCollectionCount := 0
	isEnd := false
	var tasks []domain.Task
	var errNext error

	for !isEnd {
		tasks, isEnd, errNext = collections.Next()
		if errNext != nil {
			log.Fatal(errNext, "Getting next part is fail")
		}
		if isEnd {
			break
		}

		for _, task := range tasks {
			if _, exist := allTasks[task.Id]; exist {
				assert.Fail(t, fmt.Sprintf("Task was founded in previous time"))
			}
			allTasks[task.Id] = task
		}
		actualCollectionCount++

		var isFound bool

		expectedCountTaskOnIteration, isFound = find(expectedCountTaskOnIteration, len(tasks))
		if !isFound {
			assert.Fail(t, fmt.Sprintf("Founded count of task in for collection is not correct"))
		}
	}

	assert.Equal(
		t,
		expectedCollectionCount,
		actualCollectionCount,
		"Founded count of collection is not correct",
	)

	assert.Equal(
		t,
		countAllTask,
		len(allTasks),
		"Founded count of all task is not correct",
	)
}

func find(a []int, x int) ([]int, bool) {
	for i, n := range a {
		if x == n {
			return append(a[:i], a[i+1:]...), true
		}
	}
	return a, false
}

// Testing race condition in case parallel access to tasks
//func Test_FindBySecToExecTimeRaceCondition(t *testing.T) {
//	clear()
//
//	expectedTaskCount := 1000
//	workersCount := 20
//	countRequestForWorker := 3
//
//	fixtureTasks := NewFixtureTaskBuilder(appInstanceId).
//		AddTasksNotTaken(-4, 500).
//		AddTasksTakenByBrokenInstance(-5, 500).
//		GetTasks()
//
//	loadFixtures(fixtureTasks)
//
//	workersDone := sync.WaitGroup{}
//	workersDone.Add(workersCount)
//
//	startWorkers := make(chan struct{})
//	foundTasks := make(chan domain.Task, expectedTaskCount*5)
//
//	for worker := 0; worker < workersCount; worker++ {
//		workerNum := worker
//		go func() {
//			t.Log(fmt.Sprintf("Start worker:%d", workerNum))
//			defer workersDone.Done()
//			<-startWorkers
//
//			for i := 0; i < countRequestForWorker; i++ {
//				tasks, err := repository.FindBySecToExecTime(0)
//				if err != nil {
//					log.Fatal(err, "Error while get tasks")
//				}
//				t.Log(fmt.Sprintf(
//					"WorkerNum:%d get %d count of tasks. Connections - InUse:%d Idle:%d",
//					workerNum,
//					len(tasks),
//					db.Stats().InUse,
//					db.Stats().Idle,
//				))
//
//				for _, task := range tasks {
//					foundTasks <- task
//				}
//			}
//		}()
//	}
//
//	close(startWorkers)
//
//	workersDone.Wait()
//	close(foundTasks)
//
//	var taskIdsFound = make([]domain.Task, 0, expectedTaskCount*5)
//	for taskFound := range foundTasks {
//		taskIdsFound = append(taskIdsFound, taskFound)
//	}
//	sortTaskById(taskIdsFound)
//
//	for i := 1; i < expectedTaskCount; i++ {
//		assert.True(t,
//			fixtureTasks[i].Id == taskIdsFound[i].Id &&
//				fixtureTasks[i].ExecTime == taskIdsFound[i].ExecTime,
//			"Tasks is not equal",
//		)
//	}
//	assert.Len(t, fixtureTasks, expectedTaskCount, "Count of tasks is not equal")
//}

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
				errCreate := repository.Create(domain.Task{ExecTime: time.Now().Unix()}, test.isTaken)
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
	_, errTruncateTask := db.Exec("delete from task")
	if errTruncateTask != nil {
		log.Fatal(errTruncateTask, "Error clear task")
	}
	_, errTruncateCollection := db.Exec("delete from collection")
	if errTruncateCollection != nil {
		log.Fatal(errTruncateCollection, "Error clear collection")
	}
}
