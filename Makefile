creating_and_deleting:
	go run cmd/benchmark/*.go -test_name=creating_and_deleting -task_count=100000

sending_and_confirmation:
	go run cmd/benchmark/*.go -test_name=sending_and_confirmation -task_count=100000
