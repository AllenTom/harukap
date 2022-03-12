package youlog

import (
	"context"
	"fmt"
	"github.com/allentom/harukap/config"
	"github.com/project-xpolaris/youplustoolkit/youlog"
	"github.com/rs/xid"
	"time"
)

type Plugin struct {
	Logger *youlog.LogClient
}

func (p *Plugin) OnInit(config *config.Provider) error {
	p.Logger = &youlog.LogClient{}
	instance := config.Manager.GetString("log.youlog.instance")
	application := config.Manager.GetString("log.youlog.application")
	addr := config.Manager.GetString("log.youlog.addr")
	remote := config.Manager.GetBool("log.youlog.remote")
	if len(instance) == 0 {
		instance = fmt.Sprintf("%s_%s", application, xid.New().String())
	}
	p.Logger.Init(addr, application, instance)
	p.Logger.Remote = remote
	if remote {
		timeout, _ := context.WithTimeout(context.Background(), 10*time.Second)
		err := p.Logger.Connect(timeout)
		if err != nil {
			return err
		}
	}
	return nil
}
