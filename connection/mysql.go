package connection

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type Options struct {
	User            string
	Password        string
	Host            string
	DbName          string
	SetMaxIdleConns int
	SetMaxOpenConns int
}

func NewMysqlClient(options Options) *sql.DB {
	dataSourceName := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=utf8",
		options.User,
		options.Password,
		options.Host,
		options.DbName,
	)
	Client, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		panic(err)
	}

	if err := Client.Ping(); err != nil {
		panic(err)
	}

	if options.SetMaxIdleConns > 0 {
		Client.SetMaxIdleConns(options.SetMaxIdleConns)
	}

	if options.SetMaxOpenConns > 0 {
		Client.SetMaxOpenConns(options.SetMaxOpenConns)
	}

	return Client
}
