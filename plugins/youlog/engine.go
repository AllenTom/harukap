package youlog

import (
	"context"
)

type LogData struct {
	Application string `json:"application"`
	Instance    string `json:"instance"`
	Scope       string `json:"scope"`
	Extra       string `json:"extra"`
	Message     string `json:"message"`
	Level       int64  `json:"level"`
	Time        int64  `json:"time"`
}
type LogEngine interface {
	Init() error
	WriteLog(context context.Context, scope *Scope, message string, level int64) error
}
