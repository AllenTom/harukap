package datasource

import (
	"errors"
	"fmt"
	"github.com/allentom/harukap"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Plugin struct {
	DataSource  Datasource
	Dialector   gorm.Dialector
	OnConnected func(db *gorm.DB)
	DBS         map[string]*gorm.DB
}

func (p *Plugin) OnInit(e *harukap.HarukaAppEngine) error {
	configure := e.ConfigProvider.Manager
	dataSourceList := configure.GetStringMap("datasource")
	p.DBS = make(map[string]*gorm.DB)
	for source := range dataSourceList {
		prefix := fmt.Sprintf("datasource.%s", source)
		datasourceType := configure.GetString(fmt.Sprintf("%s.type", prefix))
		var dbSource Datasource
		if datasourceType == "sqlite" {
			dbSource = &Sqlite{}
		} else if datasourceType == "mysql" {
			dbSource = &Mysql{}
		} else {
			return errors.New("unknown datasource type")
		}
		dia, err := dbSource.OnGetDialector(e.ConfigProvider.Manager, prefix)
		if err != nil {
			return err
		}
		db, err := gorm.Open(dia, &gorm.Config{})
		if err != nil {
			return err
		}
		if p.OnConnected != nil {
			p.OnConnected(db)
		}
		p.DBS[source] = db
	}
	return nil
}

type Datasource interface {
	OnGetDialector(config *viper.Viper, prefix string) (gorm.Dialector, error)
}
