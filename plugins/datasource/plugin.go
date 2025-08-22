package datasource

import (
	"errors"
	"fmt"
	"time"

	"github.com/allentom/harukap"
	util "github.com/allentom/harukap/utils"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Plugin struct {
	DataSource  Datasource
	Dialector   gorm.Dialector
	OnConnected func(db *gorm.DB)
	DBS         map[string]*gorm.DB
}

// 验证数据源配置
func validateDataSourceConfig(config *viper.Viper, prefix string) error {
	requiredFields := []string{"type"}
	for _, field := range requiredFields {
		if !config.IsSet(fmt.Sprintf("%s.%s", prefix, field)) {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

func (p *Plugin) OnInit(e *harukap.HarukaAppEngine) error {
	initLogger := e.LoggerPlugin.Logger.NewScope("DatasourcePlugin")
	initLogger.Info("initializing datasource plugin")

	configure := e.ConfigProvider.Manager
	dataSourceList := configure.GetStringMap("datasource")
	if len(dataSourceList) == 0 {
		return errors.New("no datasource configuration found")
	}

	p.DBS = make(map[string]*gorm.DB)

	for source := range dataSourceList {
		initLogger.Info("initializing datasource", "source", source)
		prefix := fmt.Sprintf("datasource.%s", source)

		// 验证配置
		if err := validateDataSourceConfig(configure, prefix); err != nil {
			return fmt.Errorf("invalid configuration for datasource %s: %v", source, err)
		}

		datasourceType := configure.GetString(fmt.Sprintf("%s.type", prefix))
		var dbSource Datasource

		// 输出配置（掩码敏感信息）
		fields := map[string]interface{}{
			"source": source,
			"type":   datasourceType,
		}

		switch datasourceType {
		case "sqlite":
			dbSource = &Sqlite{}
			fields["path"] = configure.GetString(fmt.Sprintf("%s.path", prefix))
		case "mysql":
			dbSource = &Mysql{}
			fields["host"] = configure.GetString(fmt.Sprintf("%s.host", prefix))
			fields["port"] = configure.GetString(fmt.Sprintf("%s.port", prefix))
			fields["database"] = configure.GetString(fmt.Sprintf("%s.database", prefix))
			user := configure.GetString(fmt.Sprintf("%s.username", prefix))
			pwd := configure.GetString(fmt.Sprintf("%s.password", prefix))
			fields["username"] = util.MaskKeepHeadTail(user, 1, 1)
			fields["password"] = util.MaskKeepHeadTail(pwd, 1, 2)
		default:
			return fmt.Errorf("unknown datasource type: %s", datasourceType)
		}

		initLogger.WithFields(fields).Info("datasource config")

		dia, err := dbSource.OnGetDialector(e.ConfigProvider.Manager, prefix)
		if err != nil {
			return fmt.Errorf("failed to create dialector for %s: %v", source, err)
		}

		// 配置 GORM
		gormConfig := &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
			NowFunc: func() time.Time {
				return time.Now().Local()
			},
		}

		db, err := gorm.Open(dia, gormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to database %s: %v", source, err)
		}

		// 配置连接池
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("failed to get database instance for %s: %v", source, err)
		}

		// 设置连接池参数
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)

		if p.OnConnected != nil {
			p.OnConnected(db)
		}

		p.DBS[source] = db
		initLogger.Info("successfully initialized datasource", "source", source)
	}

	return nil
}

func (p *Plugin) GetPluginConfig() map[string]interface{} {
	cfg := map[string]interface{}{}
	list := map[string]string{}
	for name := range p.DBS {
		list[name] = "connected"
	}
	cfg["databases"] = list
	return cfg
}

type Datasource interface {
	OnGetDialector(config *viper.Viper, prefix string) (gorm.Dialector, error)
}
