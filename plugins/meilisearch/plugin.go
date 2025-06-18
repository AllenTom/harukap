package meilisearch

import (
	"github.com/allentom/harukap"
	"github.com/meilisearch/meilisearch-go"
)

type Plugin struct {
	Client     meilisearch.ServiceManager
	OnComplete func()
}

func (p *Plugin) OnInit(e *harukap.HarukaAppEngine) error {
	initLogger := e.LoggerPlugin.Logger.NewScope("MeiliSearchPlugin")
	initLogger.Info("init MeiliSearch plugin")
	configure := e.ConfigProvider.Manager
	enable := configure.GetBool("meilisearch.enable")
	if !enable {
		initLogger.Info("meilisearch is disabled")
		return nil
	}
	host := configure.GetString("meilisearch.host")
	apiKey := configure.GetString("meilisearch.apiKey")
	initLogger.WithFields(map[string]interface{}{
		"host": host,
	}).Info("init meilisearch client")
	p.Client = meilisearch.New(host, meilisearch.WithAPIKey(apiKey))
	initLogger.Info("test meilisearch connection")
	status, err := p.Client.Health()
	if err != nil {
		return err
	}
	initLogger.Info("meilisearch connection success ", "status = ", status.Status)
	if p.OnComplete != nil {
		p.OnComplete()
	}
	return err
}
