package mysql

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	//"os"
)

//const (
//	mysql_users_username = "mysql_users_username"
//	mysql_users_pass     = "mysql_users_pass"
//	mysql_users_host     = "mysql_users_host"
//	mysql_users_scheme   = "mysql_users_scheme"
//)

var (
	Client *sql.DB

//	username = os.Getenv(mysql_users_username)
//	pass     = os.Getenv(mysql_users_pass)
//	host     = os.Getenv(mysql_users_host)
//	scheme   = os.Getenv(mysql_users_scheme)
)

func init() {
	datasourceName := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=utf8",
		"root",
		"secret",
		"localhost",
		"users_db",
	)
	var err error
	Client, err = sql.Open("mysql", datasourceName)
	if err != nil {
		panic(err)
	}
	if err := Client.Ping(); err != nil {
		panic(err)
	}
	log.Println("db successfully connected")
}
