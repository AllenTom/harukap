package datasource

import (
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Mysql struct {
}

func (s *Mysql) OnGetDialector(config *viper.Viper) (gorm.Dialector, error) {
	username := config.GetString("mysql.username")
	password := config.GetString("mysql.password")
	host := config.GetString("mysql.host")
	port := config.GetString("mysql.port")
	database := config.GetString("mysql.database")
	connectString := fmt.Sprintf("%s:%s@(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		username,
		password,
		host,
		port,
		database,
	)
	return mysql.Open(connectString), nil
}
