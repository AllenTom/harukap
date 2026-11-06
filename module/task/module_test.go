package task

import (
	"testing"
	"time"
)

type stubTask struct {
	BaseTask
	output interface{}
}

func (t *stubTask) Stop() error                  { return nil }
func (t *stubTask) Start() error                 { return nil }
func (t *stubTask) Output() (interface{}, error) { return t.output, nil }

func newStub(id, typ string, children ...Task) *stubTask {
	t := &stubTask{
		BaseTask: BaseTask{
			Id:          id,
			Type:        typ,
			Status:      StatusTextRunning,
			Created:     time.Unix(1700000000, 0).UTC(),
			SubTaskList: children,
		},
		output: map[string]int{"current": 1, "total": 2},
	}
	return t
}

func TestSerializerTemplateListOmitsSubtree(t *testing.T) {
	grandchild := newStub("c", "Leaf")
	child := newStub("b", "CreateVideo", grandchild)
	root := newStub("a", "ScanLibrary", child)

	mod := NewTaskModule()
	mod.Pool.AddTask(root)

	raw, err := mod.SerializerTemplateList()
	if err != nil {
		t.Fatal(err)
	}
	list := raw.([]interface{})
	if len(list) != 1 {
		t.Fatalf("want 1 root task, got %d", len(list))
	}
	tp := list[0].(*Template)
	if !tp.HasSubTask || tp.SubTaskCount != 1 {
		t.Fatalf("list summary: has=%v count=%d", tp.HasSubTask, tp.SubTaskCount)
	}
	if len(tp.SubTask) != 0 {
		t.Fatalf("list must not expand subTask, got %d", len(tp.SubTask))
	}
}

func TestSerializerTemplateExpandsOneLevel(t *testing.T) {
	grandchild := newStub("c", "Leaf")
	child := newStub("b", "CreateVideo", grandchild)
	root := newStub("a", "ScanLibrary", child)

	mod := NewTaskModule()
	raw, err := mod.SerializerTemplate(root)
	if err != nil {
		t.Fatal(err)
	}
	tp := raw.(*Template)
	if len(tp.SubTask) != 1 {
		t.Fatalf("detail should expand one child, got %d", len(tp.SubTask))
	}
	childTpl := tp.SubTask[0]
	if childTpl.Id != "b" {
		t.Fatalf("child id = %s", childTpl.Id)
	}
	if !childTpl.HasSubTask || childTpl.SubTaskCount != 1 {
		t.Fatalf("child summary: has=%v count=%d", childTpl.HasSubTask, childTpl.SubTaskCount)
	}
	if len(childTpl.SubTask) != 0 {
		t.Fatalf("detail must not recurse into grandchildren, got %d", len(childTpl.SubTask))
	}
}
