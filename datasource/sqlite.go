package datasource

import (
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Sqlite struct {
}

func (s *Sqlite) OnGetDialector(config *viper.Viper) (gorm.Dialector, error) {
	path := config.GetString("sqlite.path")
	return sqlite.Open(path), nil
}
