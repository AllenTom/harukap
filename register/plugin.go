package register

import (
	"github.com/allentom/harukap"
)

type RegisterPlugin struct {
}

func (p *RegisterPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	client := RegisterClient{
		Endpoints: e.ConfigProvider.Manager.GetStringSlice("register.endpoints"),
	}
	err := client.Init()
	if err != nil {
		return err
	}
	return RegisterFromFile(e.ConfigProvider.Manager.GetString("register.config"), &client)
}
