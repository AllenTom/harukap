package nsfwcheck

import (
	"fmt"
	"github.com/allentom/harukap"
)

type Plugin struct {
	Client *Client
	Enable bool
}

func NewPlugin() *Plugin {
	return &Plugin{}
}
func (p *Plugin) OnInit(e *harukap.HarukaAppEngine) error {
	initLogger := e.LoggerPlugin.Logger.NewScope("NSFWCheckPlugin")
	initLogger.Info("init NSFW Check plugin")
	configure := e.ConfigProvider.Manager
	enable := configure.GetBool("nsfwcheck.enable")
	p.Enable = enable
	if !enable {
		initLogger.Info("nsfwcheck is disabled")
		return nil
	}
	host := configure.GetString("nsfwcheck.host")
	initLogger.Info(fmt.Sprintf("init nsfwcheck client, host = %s", host))
	p.Client = NewClient(host)
	infoResponse, err := p.Client.Info()
	if err != nil {
		return err
	}
	if infoResponse.Success != true {
		return fmt.Errorf("info response success is false")
	}
	initLogger.Info("nsfwcheck connection success")
	return nil
}
