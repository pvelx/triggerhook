package triggerHook

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pvelx/triggerHook/connection"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/sender_service"
	"github.com/stretchr/testify/assert"
	"log"
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
)

//func TestMain(m *testing.M) {
//	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
//	pool, err := dockertest.NewPool("")
//	if err != nil {
//		log.Fatalf("Could not connect to docker: %s", err)
//	}
//
//	var resource *dockertest.Resource
//
//	//Delete old container if one was not deleted in previous test due to fatal error
//	if err = pool.RemoveContainerByName(containerName); err != nil {
//		log.Fatalf("Does not deleted: %s", err)
//	}
//
//	opts := dockertest.RunOptions{
//		Name:       containerName,
//		Repository: "mysql",
//		Tag:        "8.0",
//		Env: []string{
//			"MYSQL_ROOT_PASSWORD=" + password,
//			"MYSQL_DATABASE=" + dbName,
//			"MYSQL_PASSWORD=" + password,
//		},
//		ExposedPorts: []string{"3306"},
//		PortBindings: map[docker.Port][]docker.PortBinding{
//			"3306": {
//				{HostIP: "0.0.0.0", HostPort: port},
//			},
//		},
//	}
//
//	var errRun error
//	resource, errRun = pool.RunWithOptions(&opts)
//	if errRun != nil {
//		log.Fatalf("Could not start resource: %s", err)
//	}
//
//	dsn = fmt.Sprintf(dsn, user, password, port, dbName)
//
//	if err := pool.Retry(func() error {
//		var err error
//		db, err = sql.Open(dialect, dsn)
//		if err != nil {
//			return err
//		}
//		return db.Ping()
//	}); err != nil {
//		log.Fatalf("Could not connect to docker: %s", err)
//	}
//
//	db.SetMaxIdleConns(idleConn)
//	db.SetMaxOpenConns(maxConn)
//
//	code := m.Run()
//
//	if err := pool.Purge(resource); err != nil {
//		log.Fatalf("Could not purge resource: %s", err)
//	}
//
//	os.Exit(code)
//}

func TestExample(t *testing.T) {

	actualAllTasksCount := 0
	triggerHook := Build(Config{
		Connection: connection.Options{
			User:     "root",
			Password: "secret",
			Host:     "127.0.0.1:3306",
			DbName:   "test_db",
		},
		SenderServiceOptions: sender_service.Options{
			Transport: func(task domain.Task) {
				actualAllTasksCount++
				assert.Equal(t, time.Now().Unix(), task.ExecTime,
					"time exec of the task is not current time")
			},
		},
	})

	go func() {
		if err := triggerHook.Run(); err != nil {
			log.Fatal(err)
		}
	}()
	//it takes time for run trigger hook
	time.Sleep(time.Second)

	inputData := []struct {
		tasksCount       int
		relativeExecTime int64
	}{
		{100, -2},
		{200, 0},
		{210, 1},
		{220, 5},
		{230, 7},
		{240, 10},
		{250, 12},
	}
	expectedAllTasksCount := 0

	for _, current := range inputData {
		expectedAllTasksCount = expectedAllTasksCount + current.tasksCount
		current := current
		go func() {
			for i := 0; i < current.tasksCount; i++ {
				execTime := time.Now().Add(time.Duration(current.relativeExecTime) * time.Second).Unix()
				if err := triggerHook.Create(&domain.Task{
					ExecTime: execTime,
				}); err != nil {
					fmt.Println(err)
					t.Fatal(err)
				}
			}
		}()
	}

	// it takes time to process the most deferred tasks
	time.Sleep(13 * time.Second)

	assert.Equal(t, expectedAllTasksCount, actualAllTasksCount, "count tasks is not correct")
}
