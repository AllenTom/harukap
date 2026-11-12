package youlog

import (
	"context"
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"time"
)

type FluentdEngineConfig struct {
	Url string
}
type FluentdEngine struct {
	Config *FluentdEngineConfig
	client *resty.Client
}

func (e *FluentdEngine) Init() error {
	e.client = resty.New()
	return nil
}

func (e *FluentdEngine) WriteLog(context context.Context, scope *Scope, message string, level int64) error {
	extra, err := json.Marshal(scope.Fields)
	data := &LogData{
		Application: scope.LogClient.Application,
		Instance:    scope.LogClient.Instance,
		Scope:       scope.Name,
		Extra:       string(extra),
		Message:     message,
		Level:       level,
		Time:        time.Now().UnixMilli(),
	}
	_, err = e.client.NewRequest().SetBody(data).SetContext(context).Post(e.Config.Url)
	return err
}
