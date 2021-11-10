package youlog

import (
	"context"
	"errors"
	"fmt"
	"github.com/allentom/harukap/config"
	util "github.com/allentom/harukap/utils"
	"github.com/project-xpolaris/youplustoolkit/youlog"
	"github.com/rs/xid"
	"github.com/spf13/viper"
	"time"
)

type Plugin struct {
	Logger *youlog.LogClient
}

func (p *Plugin) OnInit(config *config.Provider) error {
	p.Logger = &youlog.LogClient{}
	util.SetDefaultWithKeys(config.Manager, "log.youlog.application", "application")
	application := config.Manager.GetString("log.youlog.application")
	if len(application) == 0 {
		return errors.New("no log [application] config exist")
	}
	util.SetDefaultWithKeys(config.Manager, "log.youlog.instance", "instance")
	instance := config.Manager.GetString("log.youlog.instance")
	if len(instance) == 0 {
		instance = fmt.Sprintf("%s_%s", application, xid.New().String())
	}
	addr := config.Manager.GetString("log.youlog.addr")
	remote := config.Manager.GetBool("log.youlog.remote")
	viper.SetDefault("log.youlog.retry", 3000)
	retry := config.Manager.GetInt("log.youlog.retry")
	p.Logger.Init(addr, application, instance)
	if remote {
		timeout, _ := context.WithTimeout(context.Background(), 10*time.Second)
		err := p.Logger.Connect(timeout)
		if err != nil {
			return err
		}
		p.Logger.StartDaemon(retry)
	}
	return nil
}
