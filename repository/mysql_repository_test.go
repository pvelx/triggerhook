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
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
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

	repository := NewRepository(db, appInstanceId, ErrorHandler{}, nil)

	if err := repository.Up(); err != nil {
		log.Fatalf("Up schema is fail: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

// Testing race condition in case parallel access to tasks
func Test_FindBySecToExecTimeRaceCondition(t *testing.T) {
	clear()
	loadFixtures("data_1")

	repository := NewRepository(db, appInstanceId, ErrorHandler{}, nil)

	expectedTaskCount := 35060
	workersCount := 10

	workersDone := sync.WaitGroup{}
	workersDone.Add(workersCount)

	startWorkers := make(chan bool)
	foundTasks := make(chan domain.Task, expectedTaskCount*2)

	result, err := repository.FindBySecToExecTime(5)
	if err != nil {
		log.Fatal(err, "Error while get tasks")
	}
	allTasks := make(map[string]domain.Task)

	foundedCountOfTasks := 0
	for worker := 0; worker < workersCount; worker++ {
		workerNum := worker
		go func() {
			defer workersDone.Done()
			<-startWorkers

			for {
				tasks, isEnd, err := result.Next()
				if err != nil {
					panic("Cannot get tasks for doing")
					return
				}
				if isEnd {
					break
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

	var tasks = make([]domain.Task, 0, expectedTaskCount*2)
	for task := range foundTasks {
		tasks = append(tasks, task)
		if _, exist := allTasks[task.Id]; exist {
			assert.Fail(t, fmt.Sprintf("The task already was founded in previous time"))
		}
		allTasks[task.Id] = task
		foundedCountOfTasks++
	}

	assert.Equal(t, expectedTaskCount, foundedCountOfTasks, "Count of tasks is not equal")
}

func TestFindBySecToExecTime(t *testing.T) {
	clear()
	loadFixtures("data_2")

	repository := NewRepository(db, appInstanceId, ErrorHandler{}, nil)

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

func TestCreateRaceCondition(t *testing.T) {
	clear()

	maxCountTasksInCollection := 100
	repository := NewRepository(
		db, appInstanceId, ErrorHandler{}, &Options{maxCountTasksInCollection})

	now := time.Now().Unix()
	input := []struct {
		tasksCount       int
		isTaken          bool
		relativeExecTime int64
	}{
		{180, false, -5},
		{150, false, 0},
		{150, false, 1},
		{220, false, 2},
		{140, true, -5},
		{250, true, 0},
		{120, true, 1},
		{80, true, 2},
	}

	workersCount := 5
	workersDone := sync.WaitGroup{}
	workersDone.Add(workersCount)
	startWorkers := make(chan bool)
	for worker := 0; worker < workersCount; worker++ {
		workerNum := worker
		go func() {
			defer workersDone.Done()
			<-startWorkers
			for _, item := range input {
				t.Log(fmt.Sprintf(
					"WorkerNum:%d. Connections - InUse:%d Idle:%d",
					workerNum,
					db.Stats().InUse,
					db.Stats().Idle,
				))
				for i := 0; i < item.tasksCount; i++ {
					errCreate := repository.Create(getTaskInstance(now+item.relativeExecTime), item.isTaken)
					if errCreate != nil {
						log.Fatal(errCreate, "Error while create")
					}
				}
			}
		}()
	}

	close(startWorkers)
	workersDone.Wait()

	for i, item := range input {
		assert.Equal(t,
			item.tasksCount*workersCount,
			getCountTasksByParamsInDb(item.isTaken, now+item.relativeExecTime),
			fmt.Sprintf("Count of tasks is not correct (isTaken:%t, execTime:%d, input item: %d)",
				item.isTaken, now+item.relativeExecTime, i),
		)

		/*
			Due to concurrent access to DB may be created extra collection. It is not big problem.
			It is assumed that the number of extra collections should not exceed twice the norm.
		*/
		maxApproximatelyCountOfCollections :=
			int(float64(item.tasksCount*workersCount) / float64(maxCountTasksInCollection) * 2)

		assert.LessOrEqual(t,
			getCountCollectionsByParamsInDb(item.isTaken, now+item.relativeExecTime),
			maxApproximatelyCountOfCollections,
			fmt.Sprintf("Count of collections is not correct (isTaken:%t, execTime:%d, input item: %d)",
				item.isTaken, now+item.relativeExecTime, i),
		)
	}
}

func TestDeleteBunch(t *testing.T) {
	clear()
	loadFixtures("data_3")
	repository := NewRepository(db, appInstanceId, ErrorHandler{}, nil)

	tasksMustNotBeDeleted := []domain.Task{
		{Id: "1657bd33-83d0-4a02-ab23-288a8ea33452"},
		{Id: "19dd8fce-b3f6-4347-b6fa-d9d075a22c71"},
		{Id: "1b959ecb-6b18-48aa-8804-cc604201675e"},
		{Id: "05721684-22ec-499b-9267-6498e57d5755"},
		{Id: "087c988b-1b68-46d8-8a45-efdf916a84b1"},
		{Id: "01c796df-7e2d-4981-86b0-67eb1a7fc45b"},
		{Id: "02ab39b5-d3ed-4de4-8f1c-bd4894cbee02"},
		{Id: "a0d96c2e-46f3-11eb-9f26-5ee87590738f"},
		{Id: "a128b982-46f3-11eb-9f26-5ee87590738f"},
		{Id: "131290fe-46f4-11eb-9f26-5ee87590738f"},
		{Id: "13287d06-46f4-11eb-9f26-5ee87590738f"},
		{Id: "13763ed8-46f4-11eb-9f26-5ee87590738f"},
		{Id: "aa1de8ae-46f4-11eb-9f26-5ee87590738f"},
		{Id: "aa29fc84-46f4-11eb-9f26-5ee87590738f"},
	}

	tasksMustBeDeleted := []domain.Task{
		{Id: "0aa6c808-207d-453f-8d8f-a3775f99f05c"},
		{Id: "13cbc999-c428-4d4f-bad3-3959e747769c"},
		{Id: "02b9bc69-4fb7-4519-b51f-8e66dc48a26a"},
		{Id: "059de99b-30a5-4018-8c79-f44767580cd5"},
		{Id: "070df841-9a71-4a06-bb55-12cf096acbda"},
		{Id: "0eaadd31-71e8-4dc2-a687-ed981e1d78ad"},
		{Id: "0f798362-2e12-48df-b1ea-f78736807005"},
		{Id: "021cba07-8fa0-4111-bf5c-338b40fd856c"},
		{Id: "03070b08-df29-4ef7-8226-d628548ff0ed"},
		{Id: "03e4d83d-9c42-461f-b663-205c26950dce"},
		{Id: "058b3fd3-2a07-46b0-aa95-099aa2abbc78"},
		{Id: "05db598c-abbb-4026-b0fe-ee19289c25f2"},
		{Id: "0a62e9d9-a4b4-4053-8563-b24b8ce4bcaa"},
		{Id: "0aa88a99-9872-4cf9-aca7-ad23a45e8f60"},
		{Id: "0ba79a57-b5c7-409c-b15a-bf70222b7d13"},
		{Id: "0c1acbde-aeec-4cee-866b-bb455b98f5d5"},
		{Id: "0e0cba0f-493b-4294-a11c-e26e917d0ab7"},
		{Id: "000c1709-b8a6-4674-9e6b-7415bd9308a3"},
		{Id: "002e256b-30c6-4562-a2af-5d778a6caabd"},
		{Id: "003f5516-31e3-4f81-b819-9af1c1cc6a4a"},
		{Id: "00717d01-7d07-48ba-b007-7979a6b14940"},
		{Id: "008797b1-340f-4440-80f8-b5e76622501a"},
		{Id: "005b5228-97e2-450c-adb7-c7f2248cab46"},
		{Id: "00785c9a-9b5e-4b9d-aad1-055ca1543cce"},
		{Id: "00a0ace5-45e1-4353-a007-e23fd1d1dbff"},
		{Id: "00a60038-d738-44c7-bfe8-a6741ab0024b"},
		{Id: "01a17c46-7536-406c-bda6-3b204ddcdba0"},
		{Id: "03ffca7b-fb47-4db6-8bd9-8a8f729d8801"},
		{Id: "043158b7-8a22-4ea1-83b0-46b76a34822f"},
		{Id: "0432ff60-1fef-42f7-9d43-c116a60bfdf3"},
		{Id: "0012f748-fb2a-49cd-a455-e73c0212466a"},
		{Id: "00613adf-8104-47bc-b655-7589bef7bdd6"},
		{Id: "00c8851e-a675-4ed4-81f1-5cd94c0c5d6a"},
		{Id: "9ebdfa40-46f3-11eb-9f26-5ee87590738f"},
		{Id: "a0064d94-46f3-11eb-9f26-5ee87590738f"},
		{Id: "a0876816-46f3-11eb-9f26-5ee87590738f"},
		{Id: "12dceaa8-46f4-11eb-9f26-5ee87590738f"},
		{Id: "12f7adb6-46f4-11eb-9f26-5ee87590738f"},
		{Id: "85dc6970-46f4-11eb-9f26-5ee87590738f"},
		{Id: "86283026-46f4-11eb-9f26-5ee87590738f"},
		{Id: "86347c50-46f4-11eb-9f26-5ee87590738f"},
		{Id: "8642c4e0-46f4-11eb-9f26-5ee87590738f"},
		{Id: "86501370-46f4-11eb-9f26-5ee87590738f"},
		{Id: "a9b74ec8-46f4-11eb-9f26-5ee87590738f"},
		{Id: "aa03602e-46f4-11eb-9f26-5ee87590738f"},
		{Id: "aa10b454-46f4-11eb-9f26-5ee87590738f"},
	}

	collectionsMustBeDeleted := []int{25, 67, 107, 129}
	collectionsMustNotBeDeleted := []int{14, 37, 42, 48, 63, 74, 81, 94, 100, 110, 122, 123, 124, 125, 126, 127, 128}

	err := repository.Delete(tasksMustBeDeleted)
	if err != nil {
		log.Fatal(err, "Error while delete")
	}

	for _, task := range tasksMustNotBeDeleted {
		assert.True(t, isTaskExistInDb(task.Id), fmt.Sprintf("the task %s not found", task.Id))
	}

	for _, id := range collectionsMustNotBeDeleted {
		assert.True(t, isCollectionExistInDb(id), fmt.Sprintf("the collection %d not found", id))
	}

	for _, id := range collectionsMustBeDeleted {
		assert.False(t, isCollectionExistInDb(id), fmt.Sprintf("the collection %d was found", id))
	}
}

func TestCreate(t *testing.T) {
	clear()
	maxCountTasksInCollection := 100
	repository := NewRepository(db, appInstanceId, ErrorHandler{}, &Options{maxCountTasksInCollection})

	input := []struct {
		tasksCount       int
		isTaken          bool
		relativeExecTime int64
	}{
		{110, false, -5},
		{150, false, 0},
		{180, false, 1},
		{220, false, 2},
		{140, true, -5},
		{250, true, 0},
		{120, true, 1},
		{30, true, 2},
	}

	now := time.Now().Unix()
	for _, item := range input {
		for i := 0; i < item.tasksCount; i++ {
			errCreate := repository.Create(getTaskInstance(now+item.relativeExecTime), item.isTaken)
			if errCreate != nil {
				log.Fatal(errCreate, "Error while create")
			}
		}
	}

	for _, item := range input {
		assert.Equal(t,
			item.tasksCount,
			getCountTasksByParamsInDb(item.isTaken, now+item.relativeExecTime),
			fmt.Sprintf("Count of tasks is not correct (isTaken:%t, execTime:%d)",
				item.isTaken, now+item.relativeExecTime),
		)

		assert.Equal(t,
			int(math.Ceil(float64(item.tasksCount)/float64(maxCountTasksInCollection))),
			getCountCollectionsByParamsInDb(item.isTaken, now+item.relativeExecTime),
			fmt.Sprintf("Count of collections is not correct (isTaken:%t, execTime:%d)",
				item.isTaken, now+item.relativeExecTime),
		)
	}
}

/*
	----------------------------------------------------
	-------------------- test tools --------------------
	----------------------------------------------------
*/
func getTaskInstance(execTime int64) domain.Task {
	return domain.Task{Id: uuid.NewV4().String(), ExecTime: execTime}
}

func getCountTasksByParamsInDb(isTaken bool, execTime int64) int {
	op := "!="
	if isTaken {
		op = "="
	}
	var count int
	query := fmt.Sprintf(`SELECT count(uuid)
		FROM task
		INNER JOIN collection c on task.collection_id = c.id
		WHERE exec_time = ? AND taken_by_instance %s ?`, op)
	err := db.QueryRow(query, execTime, appInstanceId).Scan(&count)
	switch {
	case err == sql.ErrNoRows:
		return 0
	case err != nil:
		log.Fatalf("query error: %v\n", err)
	}

	return count
}

func getCountCollectionsByParamsInDb(isTaken bool, execTime int64) int {
	op := "!="
	if isTaken {
		op = "="
	}
	var count int
	query := fmt.Sprintf(`SELECT count(id) 
		FROM collection
		WHERE exec_time = ? AND taken_by_instance %s ?`, op)
	err := db.QueryRow(query, execTime, appInstanceId).Scan(&count)
	switch {
	case err == sql.ErrNoRows:
		return 0
	case err != nil:
		log.Fatalf("query error: %v\n", err)
	}

	return count
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

func isTaskExistInDb(taskId string) bool {
	var id string
	err := db.QueryRow("SELECT uuid FROM task WHERE uuid = ?", taskId).Scan(&id)

	switch {
	case err == sql.ErrNoRows:
		return false
	case err != nil:
		log.Fatalf("query error: %v\n", err)
	}

	return id == taskId
}

func isCollectionExistInDb(collectionId int) bool {
	var id int
	err := db.QueryRow("SELECT id FROM collection WHERE id = ?", collectionId).Scan(&id)

	switch {
	case err == sql.ErrNoRows:
		return false
	case err != nil:
		log.Fatalf("query error: %v\n", err)
	}

	return id == collectionId
}

func find(a []int, x int) ([]int, bool) {
	for i, n := range a {
		if x == n {
			return append(a[:i], a[i+1:]...), true
		}
	}
	return a, false
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
		if e == io.EOF {
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

	isEnd := false
	for {
		recordTask, e := readerTaskFile.Read()
		if e == io.EOF {
			isEnd = true
		}

		if !isEnd {
			values = values + "(?, ?),"
			insertTaskArgs = append(insertTaskArgs, recordTask[0], recordTask[1])
		}

		if len(insertTaskArgs) > 1000 || isEnd {
			values = values[:len(values)-len(",")]

			_, err = db.Exec(insertTask+values, insertTaskArgs...)
			if err != nil {
				panic(err)
			}

			values = ""
			insertTaskArgs = nil
		}

		if isEnd {
			break
		}
	}
}
