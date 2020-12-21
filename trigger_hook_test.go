package triggerHook

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pvelx/triggerHook/clients"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
	"log"
	"testing"
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

func TestOne(t *testing.T) {

	db = clients.NewMysqlClient("root", "secret", "127.0.0.1:3306", "test_db")
	triggerHook := Default(db)

	triggerHook.SetErrorHandler(func(eventError contracts.EventError) {
		fmt.Println("error:", eventError)
	})

	i := 0
	triggerHook.SetTransport(func(task *domain.Task) {
		i++
		if i == 1e+3 {
			i = 0
			fmt.Println("send:", task)
		}
	})

	//time.Sleep(time.Second * 3)

	if err := triggerHook.Run(); err != nil {
		log.Fatal(err)
	}

}
