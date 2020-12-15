package services

//func (s *preloadingTaskService) findToExecMock() {
//	now := time.Now().Unix()
//	var idx int64 = 0
//	countOfTasks := int64(2e+7)
//	for i := now - countOfTasks/2; i < now+countOfTasks/2; i = i + 1 {
//		s.chPreloadedTask <- domain.Task{Id: idx, ExecTime: i}
//		idx++
//	}
//
//	for {
//		ts := time.Now().Unix()
//		idx++
//		s.chPreloadedTask <- domain.Task{Id: idx, ExecTime: ts}
//		time.Sleep(time.Second)
//	}
//}
