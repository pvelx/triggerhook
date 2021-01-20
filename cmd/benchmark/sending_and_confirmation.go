package main

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/pvelx/triggerHook/error_service"
	"github.com/pvelx/triggerHook/monitoring_service"
	"github.com/pvelx/triggerHook/repository"
	"github.com/pvelx/triggerHook/util"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/pvelx/triggerHook"
	"github.com/pvelx/triggerHook/connection"
	"github.com/pvelx/triggerHook/contracts"
	"github.com/pvelx/triggerHook/domain"
)

func upInitialState(taskCount int) {

	fmt.Println("\nup initial state")
	preparingBar := pb.StartNew(taskCount)

	conn := connection.NewMysqlClient(connection.Options{
		User:     "root",
		Password: "secret",
		Host:     "127.0.0.1:3306",
		DbName:   "test_db",
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

	triggerHookService := triggerHook.Build(triggerHook.Config{
		Connection: connection.Options{
			User:     "root",
			Password: "secret",
			Host:     "127.0.0.1:3306",
			DbName:   "test_db",
		},
		MonitoringServiceOptions: monitoring_service.Options{
			PeriodMeasure: 100 * time.Millisecond,
			Subscriptions: map[contracts.Topic]func(event contracts.MeasurementEvent){
				contracts.SpeedOfConfirmation: func(event contracts.MeasurementEvent) {
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
