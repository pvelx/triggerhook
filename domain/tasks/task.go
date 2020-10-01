package tasks

type Task struct {
	Id                    int64  `json:"id"`
	DeletedAt             string `json:"deleted_at"`                //Время удаления задания
	NextExecTime          int64  `json:"next_exec_time"`            //Время следующего запуска задачи
	CompletedQuantity     int64  `json:"completed_quantity"`        //Количество выполненных заданий
	PlannedQuantity       int64  `json:"planned_quantity"`          //Полное количестов заданий к выполениню
	FormulaCalcOfNextTask string `json:"formula_calc_of_next_task"` //Формула расчета следующего задания
}

type Tasks []Task
