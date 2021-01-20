package connection

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/imdario/mergo"
)

type Options struct {
	User         string
	Password     string
	Host         string
	DbName       string
	MaxIdleConns int
	MaxOpenConns int
}

func NewMysqlClient(options Options) *sql.DB {

	if err := mergo.Merge(&options, Options{
		MaxOpenConns: 25,
		MaxIdleConns: 25,
	}); err != nil {
		panic(err)
	}

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

	if options.MaxIdleConns > 0 {
		Client.SetMaxIdleConns(options.MaxIdleConns)
	}

	if options.MaxOpenConns > 0 {
		Client.SetMaxOpenConns(options.MaxOpenConns)
	}

	return Client
}
