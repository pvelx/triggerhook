package main

import (
	"flag"
	"fmt"
	"github.com/pvelx/triggerhook/repository"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/pvelx/triggerhook/connection"
)

var (
	mysqlUser     = os.Getenv("DATABASE_USER")
	mysqlPassword = os.Getenv("DATABASE_PASSWORD")
	mysqlHost     = os.Getenv("DATABASE_HOST")
	mysqlDbName   = os.Getenv("DATABASE_NAME")
)

func clear() {
	conn := connection.New(&connection.Options{
		User:     mysqlUser,
		Password: mysqlPassword,
		Host:     mysqlHost,
		DbName:   mysqlDbName,
	})
	repository.New(conn, "", nil, nil).Up()

	if _, err := conn.Exec("DELETE FROM task"); err != nil {
		log.Fatal(err)
	}
	if _, err := conn.Exec("DELETE FROM collection"); err != nil {
		log.Fatal(err)
	}
	conn.Close()
}

func main() {
	testName := flag.String("test_name", "creating_and_deleting", "max rate creating/deleting tasks")
	taskCount := flag.Int("task_count", 1000000, "count of task for the test")
	flag.Parse()
	fmt.Printf("\ncount of task: %d\n", *taskCount)

	clear()
	var data [][]string
	switch *testName {
	case "creating_and_deleting":
		fmt.Println("Benchmark: max rate creating/deleting tasks")
		data = creatingAndDeleting(*taskCount)
	case "sending_and_confirmation":
		fmt.Println("Benchmark: max rate sending/confirmation tasks")
		data = sendingAndConfirmation(*taskCount)
	default:
		log.Fatal("incorrect test_name")
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Test", "Duration", "Avr rate"})
	for _, v := range data {
		table.Append(v)
	}
	fmt.Println()
	table.Render()
}
