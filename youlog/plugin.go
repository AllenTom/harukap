package youlog

import (
	"context"
	"fmt"
	"github.com/allentom/harukap/config"
	"github.com/mitchellh/mapstructure"
	"github.com/project-xpolaris/youplustoolkit/youlog"
	"github.com/rs/xid"
	"time"
)

type YouLogPluginConfig struct {
	Instance    string `json:"instance"`
	Application string `json:"application"`
	Engines     []interface{}
}
type Plugin struct {
	Logger *youlog.LogClient
	Config *YouLogPluginConfig
}

func (p *Plugin) OnInit(configProvider *config.Provider) error {
	p.Logger = &youlog.LogClient{}
	if p.Config == nil {
		p.Config = &YouLogPluginConfig{
			Application: configProvider.Manager.GetString("log.youlog.application"),
			Instance:    configProvider.Manager.GetString("log.youlog.instance"),
		}
	}
	if len(p.Config.Instance) == 0 {
		p.Config.Instance = fmt.Sprintf("%s_%s", p.Config.Application, xid.New().String())
	}
	p.Logger.Init(p.Config.Application, p.Config.Instance)
	// add engine
	if p.Config.Engines == nil {
		rawEngineConfig := configProvider.Manager.GetStringMap("log.youlog.engine")
		for _, engine := range rawEngineConfig {
			engineConfig := engine.(map[string]interface{})
			switch engineConfig["type"].(string) {
			case "logrus":
				logConf := &youlog.LogrusEngineConfig{}
				err := mapstructure.Decode(engineConfig, &logConf)
				if err != nil {
					return err
				}
				err = p.Logger.AddEngine(logConf)
				if err != nil {
					return err
				}
			case "youlogservice":
				logConf := &youlog.YouLogServiceEngineConfig{}
				err := mapstructure.Decode(engineConfig, &logConf)
				if err != nil {
					return err
				}
				err = p.Logger.AddEngine(logConf)
				if err != nil {
					return err
				}
			case "fluentd":
				logConf := &youlog.FluentdEngineConfig{}
				err := mapstructure.Decode(engineConfig, &logConf)
				if err != nil {
					return err
				}
				err = p.Logger.AddEngine(logConf)
				if err != nil {
					return err
				}
			}
		}
	} else {
		for _, engine := range p.Config.Engines {
			err := p.Logger.AddEngine(engine)
			if err != nil {
				return err
			}
		}
	}

	timeout, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := p.Logger.InitEngines(timeout)
	if err != nil {
		return err
	}
	return nil
}
