package storage

import (
	"fmt"

	"github.com/allentom/harukap"
)

type Engine struct {
	storages map[string]FileSystem
}

func (e *Engine) OnInit(engine *harukap.HarukaAppEngine) error {
	e.storages = make(map[string]FileSystem)
	manager := engine.ConfigProvider.Manager
	rawStorageConfig := manager.GetStringMapString("storage")
	for name := range rawStorageConfig {
		storageType := manager.GetString(fmt.Sprintf("storage.%s.type", name))
		if storageType == "" {
			continue
		}
		logger := engine.LoggerPlugin.Logger.NewScope("StorageEngine")
		logger.WithFields(map[string]interface{}{
			"name": name,
			"type": storageType,
		}).Info("storage config")
		switch storageType {
		case "s3":
			s3Plugin := &S3Client{
				ConfigName: name,
			}
			err := s3Plugin.OnInit(engine)
			if err != nil {
				return err
			}
			e.storages[name] = s3Plugin
		case "local":
			localPlugin := &LocalStorage{
				ConfigName: name,
			}
			err := localPlugin.OnInit(engine)
			if err != nil {
				return err
			}
			e.storages[name] = localPlugin
		default:
			return fmt.Errorf("unknown strage type: %s", storageType)
		}
	}
	return nil
}

func (e *Engine) GetStorage(name string) FileSystem {
	return e.storages[name]
}

func (e *Engine) GetPluginConfig() map[string]interface{} {
	cfg := map[string]interface{}{}
	for name, fs := range e.storages {
		_ = fs
		cfg[name] = "initialized"
	}
	return cfg
}
