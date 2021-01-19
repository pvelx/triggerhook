package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/pvelx/triggerHook/connection"
)

func clear() {
	conn := connection.NewMysqlClient(connection.Options{
		User:     "root",
		Password: "secret",
		Host:     "127.0.0.1:3306",
		DbName:   "test_db",
	})
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
