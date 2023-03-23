package deepdanbooru

import (
	"fmt"
	"github.com/allentom/harukap"
	"time"
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
	p.Enable = enable
	if !enable {
		initLogger.Info("deepdanbooru is disabled")
		return nil
	}
	host := configure.GetString("deepdanbooru.host")
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
	return nil
}
