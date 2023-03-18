package task

import (
	. "github.com/ahmetb/go-linq/v3"
	"sync"
)

type TaskPool struct {
	sync.Mutex
	Tasks []Task
}

func NewTaskPool() *TaskPool {
	return &TaskPool{
		Tasks: make([]Task, 0),
	}
}

func (p *TaskPool) RemoveTaskById(id string) {
	p.Lock()
	defer p.Unlock()
	var newTask []Task
	From(p.Tasks).WhereT(func(task Task) bool {
		return task.GetId() != id
	}).ToSlice(&newTask)
	p.Tasks = newTask
}

func (p *TaskPool) AddTask(task Task) {
	p.Lock()
	defer p.Unlock()
	p.Tasks = append(p.Tasks, task)
}

func (p *TaskPool) GetTaskWithStatus(taskType string, status string) Task {
	for _, task := range p.Tasks {
		if task.GetType() == taskType {
			if task.GetStatus() == status {
				return task
			}
		}
	}
	return nil
}

func (p *TaskPool) GetTaskById(id string) Task {
	if len(p.Tasks) == 0 {
		return nil
	}
	queue := make([]Task, 0)
	queue = append(queue, p.Tasks...)
	for len(queue) > 0 {
		task := queue[0]
		queue = queue[1:]
		if task.GetId() == id {
			return task
		}
		subTasks := task.SubTask()
		if len(subTasks) > 0 {
			queue = append(queue, subTasks...)
		}
	}
	return nil
}
