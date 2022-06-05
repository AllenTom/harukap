package task

import (
	"github.com/rs/xid"
	"time"
)

type Signal string

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
}

type BaseTask struct {
	Id      string
	Type    string
	Status  string
	Owner   string
	Err     error
	OnDone  chan Signal
	Created time.Time
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
func NewBaseTask(Type string, owner string, status string) *BaseTask {
	id := xid.New().String()
	return &BaseTask{
		Id:      id,
		Type:    Type,
		Owner:   owner,
		Status:  status,
		OnDone:  make(chan Signal),
		Created: time.Now(),
	}
}
