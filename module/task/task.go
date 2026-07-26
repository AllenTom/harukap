package task

import (
	"fmt"
	"time"

	"github.com/rs/xid"
)

type Signal string

const (
	StatusRunning = iota + 10000
	StatusDone
	StatusError
	StatusPending
	StatusStopped
)

// Canonical status strings used by Template JSON.
const (
	StatusTextPending  = "Pending"
	StatusTextRunning  = "Running"
	StatusTextDone     = "Done"
	StatusTextError    = "Error"
	StatusTextStopped  = "Stopped"
)

var StatusNameMapping map[int]string = map[int]string{
	StatusRunning: StatusTextRunning,
	StatusDone:    StatusTextDone,
	StatusError:   StatusTextError,
	StatusPending: StatusTextPending,
	StatusStopped: StatusTextStopped,
}

var (
	SignalDone = Signal("init")
)

type Task interface {
	GetId() string
	GetType() string
	GetStatus() string
	Stop() error
	Start() error
	Error() error
	GetCreated() time.Time
	Output() (interface{}, error)
	SubTask() []Task
	GetStartTime() time.Time
	GetEndTime() time.Time
	GetParentTaskId() string
}
type Wrap interface {
	SetStart()
	SetEnd()
	AbortError(err error) error
	Done()
	Start() error
}

type BaseTask struct {
	Id           string
	Type         string
	Status       string
	Owner        string
	Err          error
	OnDone       chan Signal
	Created      time.Time
	SubTaskList  []Task
	StartTime    time.Time
	EndTime      time.Time
	ParentTaskId string
}

func (t *BaseTask) GetStartTime() time.Time {
	return t.StartTime
}

func (t *BaseTask) GetEndTime() time.Time {
	return t.EndTime
}

func (t *BaseTask) GetId() string {
	return t.Id
}
func (t *BaseTask) Error() error {
	return t.Err
}
func (t *BaseTask) GetType() string {
	return t.Type
}
func (t *BaseTask) GetStatus() string {
	return t.Status
}
func (t *BaseTask) GetCreated() time.Time {
	return t.Created
}
func (t *BaseTask) SubTask() []Task {
	return t.SubTaskList
}
func (t *BaseTask) SetStart() {
	t.StartTime = time.Now()
	if t.Status == "" || t.Status == StatusTextPending {
		t.Status = StatusTextRunning
	}
}
func (t *BaseTask) SetEnd() {
	t.EndTime = time.Now()
}
func (t *BaseTask) GetParentTaskId() string {
	return t.ParentTaskId
}
func (t *BaseTask) AbortError(err error) error {
	t.Err = err
	t.EndTime = time.Now()
	t.Status = StatusTextError
	return err
}
func (t *BaseTask) Done() {
	t.Status = StatusTextDone
	t.EndTime = time.Now()
}

// MarkStopped sets status to Stopped and records end time.
func (t *BaseTask) MarkStopped() {
	t.Status = StatusTextStopped
	t.EndTime = time.Now()
}

func GetStatusText(extraMapping map[int]string, status int) string {
	if extraMapping != nil {
		if statusName, ok := extraMapping[status]; ok {
			return statusName
		}
	}
	if statusName, ok := StatusNameMapping[status]; ok {
		return statusName
	}
	return "Unknown"
}
func NewBaseTask(Type string, owner string, status string) *BaseTask {
	id := xid.New().String()
	if status == "" {
		status = StatusTextPending
	}
	return &BaseTask{
		Id:          id,
		Type:        Type,
		Owner:       owner,
		Status:      status,
		OnDone:      make(chan Signal),
		Created:     time.Now(),
		SubTaskList: []Task{},
	}
}

func NewSubTask(Type string, owner string, status string, parentId string) *BaseTask {
	id := xid.New().String()
	if status == "" {
		status = StatusTextPending
	}
	return &BaseTask{
		Id:           fmt.Sprintf("%s-%s", parentId, id),
		Type:         Type,
		Owner:        owner,
		Status:       status,
		OnDone:       make(chan Signal),
		Created:      time.Now(),
		SubTaskList:  []Task{},
		ParentTaskId: parentId,
	}
}

func RunTask(wrap Wrap) error {
	wrap.SetStart()
	err := wrap.Start()
	if err != nil {
		return wrap.AbortError(err)
	}
	wrap.Done()
	return nil
}
