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
