package datasource

import (
	"errors"
	"github.com/allentom/harukap"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Plugin struct {
	DataSource  Datasource
	Dialector   gorm.Dialector
	DB          *gorm.DB
	OnConnected func(db *gorm.DB)
}

func (p *Plugin) OnInit(e *harukap.HarukaAppEngine) error {
	var err error
	configure := e.ConfigProvider.Manager
	dataSourceName := configure.GetString("datasource")
	if dataSourceName == "sqlite" {
		p.DataSource = &Sqlite{}
	} else if dataSourceName == "mysql" {
		p.DataSource = &Mysql{}
	} else {
		return errors.New("unknown datasource type")
	}

	p.Dialector, err = p.DataSource.OnGetDialector(e.ConfigProvider.Manager)
	if err != nil {
		return err
	}
	p.DB, err = gorm.Open(p.Dialector, &gorm.Config{})
	if err != nil {
		return err
	}
	if p.OnConnected != nil {
		p.OnConnected(p.DB)
	}
	return nil
}

type Datasource interface {
	OnGetDialector(config *viper.Viper) (gorm.Dialector, error)
}
