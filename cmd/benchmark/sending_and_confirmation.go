package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/pvelx/triggerhook"
	"github.com/pvelx/triggerhook/connection"
	"github.com/pvelx/triggerhook/contracts"
	"github.com/pvelx/triggerhook/domain"
	"github.com/pvelx/triggerhook/error_service"
	"github.com/pvelx/triggerhook/monitoring_service"
	"github.com/pvelx/triggerhook/repository"
	"github.com/pvelx/triggerhook/util"
)

func upInitialState(taskCount int) {

	fmt.Println("\nup initial state")
	preparingBar := pb.StartNew(taskCount)

	conn := connection.New(&connection.Options{
		User:     mysqlUser,
		Password: mysqlPassword,
		Host:     mysqlHost,
		DbName:   mysqlDbName,
	})
	errorService := error_service.New(nil)
	repositoryService := repository.New(conn, util.NewId(), errorService, nil)

	rand.Seed(time.Now().UnixNano())

	dispersion := 300
	workersCount := 10
	now := time.Now().Unix()
	taskCountForWorker := taskCount / workersCount
	wg := sync.WaitGroup{}
	for w := 0; w < workersCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < taskCountForWorker; i++ {
				preparingBar.Add(1)
				task := domain.Task{
					Id:       util.NewId(),
					ExecTime: time.Unix(now, 0).Add(-time.Duration(rand.Intn(dispersion)) * time.Second).Unix(),
				}
				if err := repositoryService.Create(task, false); err != nil {
					log.Println(err)
				}
			}
		}()
	}
	wg.Wait()
	preparingBar.Finish()
	conn.Close()
}

func sendingAndConfirmation(taskCount int) [][]string {
	upInitialState(taskCount)

	var processed int
	wait := make(chan bool)
	var point time.Time
	var duration time.Duration
	var duration2 time.Duration
	var confirmed int

	fmt.Println("\nsending/confirmation tasks")
	preparingBar := pb.StartNew(taskCount)

	triggerHookService := triggerhook.Build(triggerhook.Config{
		Connection: connection.Options{
			User:     mysqlUser,
			Password: mysqlPassword,
			Host:     mysqlHost,
			DbName:   mysqlDbName,
		},
		MonitoringServiceOptions: monitoring_service.Options{
			PeriodMeasure: 100 * time.Millisecond,
			Subscriptions: map[contracts.Topic]func(event contracts.MeasurementEvent){
				contracts.ConfirmationRate: func(event contracts.MeasurementEvent) {
					confirmed = confirmed + int(event.Measurement)
					preparingBar.Add(int(event.Measurement))
					if confirmed == taskCount {
						duration2 = time.Since(point)
						preparingBar.Finish()
						close(wait)
					}
				},
			},
		},
	})

	go func() {
		for {
			triggerHookService.Consume().Confirm()
			processed = processed + 1
			if processed == taskCount {
				duration = time.Since(point)
			}
		}
	}()

	go func() {
		point = time.Now()
		if err := triggerHookService.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	<-wait

	return [][]string{
		{
			"Sending task",
			fmt.Sprintf("%v", duration),
			fmt.Sprintf("%f", float64(processed)/duration.Seconds()),
		},
		{
			"Confirmation task",
			fmt.Sprintf("%v", duration2),
			fmt.Sprintf("%f", float64(processed)/duration2.Seconds()),
		},
	}
}
