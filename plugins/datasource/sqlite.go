package datasource

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Sqlite struct {
}

func (s *Sqlite) OnGetDialector(config *viper.Viper, prefix string) (gorm.Dialector, error) {
	// 验证必需的配置项
	if !config.IsSet(prefix + ".path") {
		return nil, fmt.Errorf("missing required SQLite configuration: path")
	}

	path := config.GetString(prefix + ".path")

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory for SQLite database: %v", err)
	}

	// 检查文件权限
	if _, err := os.Stat(path); err == nil {
		// 文件存在，检查权限
		if err := os.Chmod(path, 0644); err != nil {
			return nil, fmt.Errorf("failed to set permissions for SQLite database: %v", err)
		}
	}

	return sqlite.Open(path), nil
}
