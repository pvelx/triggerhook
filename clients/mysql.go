package clients

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

func NewMysqlClient(user string, password string, host string, dbName string) *sql.DB {
	dataSourceName := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=utf8",
		user,
		password,
		host,
		dbName,
	)
	Client, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		panic(err)
	}

	if err := Client.Ping(); err != nil {
		panic(err)
	}

	Client.SetMaxIdleConns(30)
	Client.SetMaxOpenConns(30)

	return Client
}
