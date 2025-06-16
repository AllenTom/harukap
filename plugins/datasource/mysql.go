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
	// 验证必需的配置项
	requiredFields := []string{"username", "password", "host", "port", "database"}
	for _, field := range requiredFields {
		if !config.IsSet(prefix + "." + field) {
			return nil, fmt.Errorf("missing required MySQL configuration: %s", field)
		}
	}

	username := config.GetString(prefix + ".username")
	password := config.GetString(prefix + ".password")
	host := config.GetString(prefix + ".host")
	port := config.GetString(prefix + ".port")
	database := config.GetString(prefix + ".database")

	// 构建连接字符串
	connectString := fmt.Sprintf("%s:%s@(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username,
		password,
		host,
		port,
		database,
	)

	// 添加额外的配置参数
	if config.IsSet(prefix + ".params") {
		params := config.GetStringMapString(prefix + ".params")
		for key, value := range params {
			connectString += fmt.Sprintf("&%s=%s", key, value)
		}
	}

	return mysql.Open(connectString), nil
}
