package datasource

import (
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Mysql struct {
}

func (s *Mysql) OnGetDialector(config *viper.Viper, prefix string) (gorm.Dialector, error) {
	username := config.GetString(prefix + ".username")
	password := config.GetString(prefix + ".password")
	host := config.GetString(prefix + ".host")
	port := config.GetString(prefix + ".port")
	database := config.GetString(prefix + ".database")
	connectString := fmt.Sprintf("%s:%s@(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		username,
		password,
		host,
		port,
		database,
	)
	return mysql.Open(connectString), nil
}
