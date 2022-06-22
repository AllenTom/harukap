package register

import (
	"github.com/allentom/harukap"
	"github.com/project-xpolaris/youplustoolkit/youlog"
)

type RegisterPlugin struct {
	Config *RegisterConfig
	logger *youlog.Scope
}

func (p *RegisterPlugin) SetConfig(config RegisterConfig) {
	p.Config = &config
}
func (p *RegisterPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	p.logger = e.LoggerPlugin.Logger.NewScope("RegisterPlugin")
	if p.Config == nil {
		p.logger.Info("no config,use default config source")
		p.Config = &RegisterConfig{
			Enable:    e.ConfigProvider.Manager.GetBool("register.enable"),
			Endpoints: e.ConfigProvider.Manager.GetStringSlice("register.endpoints"),
			RegPath:   e.ConfigProvider.Manager.GetString("register.regpath"),
		}
	}
	if !p.Config.Enable {
		p.logger.Info("register plugin is disabled")
		return nil
	}
	client := RegisterClient{
		Endpoints: p.Config.Endpoints,
	}
	err := client.Init()
	if err != nil {
		return err
	}
	return RegisterFromFile(p.Config.RegPath, &client)
}
