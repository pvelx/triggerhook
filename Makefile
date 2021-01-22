creating_and_deleting:
	go run cmd/benchmark/*.go -test_name=creating_and_deleting -task_count=100000

sending_and_confirmation:
	go run cmd/benchmark/*.go -test_name=sending_and_confirmation -task_count=100000

test:
	GOMAXPROCS=4 go test ./ -v
	GOMAXPROCS=4 go test ./repository -v
	go test ./sender_service ./task_manager ./error_service \
		./prioritized_task_list ./preloader_service ./monitoring_service ./waiting_service -v
