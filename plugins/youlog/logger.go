package youlog

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
)

var DefaultScope = "Global"

const (
	LEVEL_DEBUG = 1
	LEVEL_INFO  = 2
	LEVEL_WARN  = 3
	LEVEL_ERROR = 4
	LEVEL_FATAL = 5
)

type LogClient struct {
	Logger      *logrus.Logger
	Application string
	Instance    string
	Engines     []LogEngine
}

type Scope struct {
	LogClient *LogClient
	Name      string
	Fields    Fields
}
type Fields map[string]interface{}

func (c *LogClient) Init(application string, instance string) {
	c.Application = application
	c.Instance = instance
	c.Engines = make([]LogEngine, 0)
	c.Logger = logrus.New()

	return
}
func (c *LogClient) InitEngines(context context.Context) error {
	for _, engine := range c.Engines {
		err := engine.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *LogClient) AddEngine(engineConfig interface{}) error {
	switch engineConfig.(type) {
	case *LogrusEngineConfig:
		c.Engines = append(c.Engines, &LogrusEngine{
			Config: engineConfig.(*LogrusEngineConfig),
		})
		return nil
	case *FluentdEngineConfig:
		c.Engines = append(c.Engines, &FluentdEngine{
			Config: engineConfig.(*FluentdEngineConfig),
		})
		return nil
	}
	return fmt.Errorf("unknown engine type")
}
func (c *LogClient) Info(message ...interface{}) {
	c.write(c.NewScope(DefaultScope), LEVEL_INFO, message...)
}
func (c *LogClient) Debug(message ...interface{}) {
	c.write(c.NewScope(DefaultScope), LEVEL_DEBUG, message...)
}
func (c *LogClient) Warn(message ...interface{}) {
	c.write(c.NewScope(DefaultScope), LEVEL_WARN, message...)
}
func (c *LogClient) Error(message ...interface{}) {
	c.write(c.NewScope(DefaultScope), LEVEL_ERROR, message...)
}
func (c *LogClient) Fatal(message ...interface{}) {
	c.write(c.NewScope(DefaultScope), LEVEL_FATAL, message...)
}
func (c *LogClient) WithFields(field Fields) *Scope {
	return c.NewScope(DefaultScope).WithFields(field)
}
func (c *LogClient) write(scope *Scope, level int64, message ...interface{}) {
	for _, engine := range c.Engines {
		err := engine.WriteLog(context.Background(), scope, combineStrings(message...), level)
		if err != nil {
			c.Logger.Error(err)
		}
	}
}
func (c *LogClient) NewScope(name string) *Scope {
	return &Scope{
		LogClient: c,
		Name:      name,
		Fields:    Fields{},
	}
}
func (c *Scope) SetFields(fields Fields) *Scope {
	c.Fields = fields
	return c
}
func (c *Scope) WithFields(fields Fields) *Scope {
	newScope := c.LogClient.NewScope(c.Name)
	for key, value := range fields {
		newScope.Fields[key] = value
	}
	for key, value := range c.Fields {
		newScope.Fields[key] = value
	}
	return newScope
}
func (c *Scope) write(level int64, message string) error {
	c.LogClient.write(c, level, message)
	return nil
}
func (c *Scope) getFieldsMap() map[string]interface{} {
	return c.Fields
}
func (c *Scope) Info(message ...interface{}) {
	c.LogClient.write(c, LEVEL_INFO, message...)
}
func (c *Scope) Debug(message ...interface{}) {
	c.LogClient.write(c, LEVEL_DEBUG, message...)
}
func (c *Scope) Warn(message ...interface{}) {
	c.LogClient.write(c, LEVEL_WARN, message...)
}
func (c *Scope) Error(message ...interface{}) {
	c.LogClient.write(c, LEVEL_ERROR, message...)
}
func (c *Scope) Fatal(message ...interface{}) {
	c.LogClient.write(c, LEVEL_FATAL, message...)
}

func combineStrings(message ...interface{}) string {
	result := ""
	for _, v := range message {
		result += fmt.Sprintf("%v", v)
	}
	return result
}
