package imageclassify

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
	initLogger := e.LoggerPlugin.Logger.NewScope("ImageClassifyPlugin")
	initLogger.Info("init ImageClassify plugin")
	configure := e.ConfigProvider.Manager
	enable := configure.GetBool("imageclassify.enable")
	if !enable {
		initLogger.Info("imageclassify is disabled")
		return nil
	}
	host := configure.GetString("imageclassify.host")
	initLogger.WithFields(map[string]interface{}{
		"enable": enable,
		"host":   host,
	}).Info("imageclassify config")
	initLogger.Info(fmt.Sprintf("init imageclassify client, host = %s", host))
	p.Client = NewClient(host)
	infoResponse, err := p.Client.Info()
	if err != nil {
		return err
	}
	if infoResponse.Success != true {
		return fmt.Errorf("info response success is false")
	}
	initLogger.Info("imageclassify connection success")
	p.Enable = enable
	return nil
}

func (p *Plugin) GetPluginConfig() map[string]interface{} {
	url := ""
	if p.Client != nil {
		url = p.Client.BaseUrl
	}
	return map[string]interface{}{
		"enable": p.Enable,
		"url":    url,
	}
}
