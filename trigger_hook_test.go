package triggerHook

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pvelx/triggerHook/connection"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"github.com/pvelx/triggerHook/error_service"
	"github.com/pvelx/triggerHook/sender_service"
	"log"
	"math/rand"
	"path"
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

var (
	baseFormat = "%s MESSAGE:%s METHOD:%s FILE:%s:%d EXTRA:%v\n"
	formats    = map[contracts.Level]string{
		contracts.LevelDebug: "DEBUG:" + baseFormat,
		contracts.LevelError: "ERROR:" + baseFormat,
		contracts.LevelFatal: "FATAL:" + baseFormat,
	}
)

func TestExample(t *testing.T) {

	i := 0
	transport := func(task domain.Task) {
		i++
		if i == 1e+4 {
			i = 0
			fmt.Println("Send:", task)
		}
	}

	eventHandlers := make(map[contracts.Level]func(event contracts.EventError))
	for level, format := range formats {
		format := format
		level := level
		eventHandlers[level] = func(event contracts.EventError) {
			_, shortMethod := path.Split(event.Method)
			_, shortFile := path.Split(event.File)
			fmt.Printf(
				format,
				event.Time.Format("2006-01-02 15:04:05.000"),
				event.EventMessage,
				shortMethod,
				shortFile,
				event.Line,
				event.Extra,
			)
		}
	}

	triggerHook := Build(Config{
		Connection: connection.Options{
			User:     "root",
			Password: "secret",
			Host:     "127.0.0.1:3306",
			DbName:   "test_db",
		},
		ErrorServiceOptions: error_service.Options{
			Debug:         false,
			EventHandlers: eventHandlers,
		},
		SenderServiceOptions: sender_service.Options{
			Transport: transport,
		},
	})

	rand.Seed(time.Now().UnixNano())

	for w := 0; w < 10; w++ {
		go func() {
			for {
				randomExecTime := time.Now().Add(time.Duration(rand.Intn(300)) * time.Second).Unix()
				err := triggerHook.Create(&domain.Task{
					ExecTime: randomExecTime,
				})
				if err != nil {
					fmt.Println(err)
				}
			}
		}()
	}

	if err := triggerHook.Run(); err != nil {
		log.Fatal(err)
	}
}
