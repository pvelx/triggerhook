package prioritized_task_list

type priorityQueue []*item

type item struct {
	task     interface{}
	priority int64
	index    int
}

func (p priorityQueue) Len() int {
	return len(p)
}

func (p priorityQueue) Less(i, j int) bool {
	return p[i].priority < p[j].priority
}

func (p priorityQueue) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].index = i
	p[j].index = j
}

func (p *priorityQueue) Push(x interface{}) {
	n := len(*p)
	item := x.(*item)
	item.index = n
	*p = append(*p, item)
}

func (p *priorityQueue) Pop() interface{} {
	old := *p
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*p = old[0 : n-1]
	return item
}
