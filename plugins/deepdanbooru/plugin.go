package deepdanbooru

import (
	"fmt"
	"time"

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
	initLogger := e.LoggerPlugin.Logger.NewScope("DeepdanbooruPlugin")
	initLogger.Info("init deepdanbooru plugin")
	configure := e.ConfigProvider.Manager
	enable := configure.GetBool("deepdanbooru.enable")
	if !enable {
		initLogger.Info("deepdanbooru is disabled")
		return nil
	}
	host := configure.GetString("deepdanbooru.host")
	initLogger.WithFields(map[string]interface{}{
		"enable": enable,
		"host":   host,
	}).Info("deepdanbooru config")
	initLogger.Info(fmt.Sprintf("init deepdanbooru client, host = %s", host))
	p.Client = NewClient(&Config{
		Url: host,
	})
	timeout := configure.GetInt("deepdanbooru.timeout")
	if timeout == 0 {
		timeout = 10000 // default 10s
	}
	p.Client.SetTimeout(time.Duration(timeout) * time.Millisecond)
	infoResponse, err := p.Client.Info()
	if err != nil {
		return err
	}
	if infoResponse.Success != true {
		return fmt.Errorf("info response success is false")
	}
	initLogger.Info("deepdanbooru connection success")
	p.Enable = enable
	return nil
}

func (p *Plugin) GetPluginConfig() map[string]interface{} {
	url := ""
	if p.Client != nil && p.Client.conf != nil {
		url = p.Client.conf.Url
	}
	return map[string]interface{}{
		"enable": p.Enable,
		"url":    url,
	}
}
