package youlog

import (
	"context"
	"github.com/sirupsen/logrus"
)

type LogrusEngineConfig struct {
}
type LogrusEngine struct {
	Logger    *logrus.Logger
	logClient *LogClient
	Config    *LogrusEngineConfig
}

func (e *LogrusEngine) Init() error {
	e.Logger = logrus.New()
	return nil
}

func (e *LogrusEngine) WriteLog(context context.Context, scope *Scope, message string, level int64) error {
	fields := logrus.Fields{}
	for key, value := range scope.Fields {
		fields[key] = value
	}
	switch level {
	case LEVEL_DEBUG:
		e.Logger.WithFields(fields).Debug(message)
	case LEVEL_INFO:
		e.Logger.WithFields(fields).Info(message)
	case LEVEL_WARN:
		e.Logger.WithFields(fields).Warn(message)
	case LEVEL_ERROR:
		e.Logger.WithFields(fields).Error(message)
	case LEVEL_FATAL:
		e.Logger.WithFields(fields).Fatal(message)
	}
	return nil
}
