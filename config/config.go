package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Provider struct {
	Manager    *viper.Viper
	OnLoaded   func(provider *Provider)
	ConfigPath string
}

func NewProvider(OnLoaded func(provider *Provider), ConfigPath string) (*Provider, error) {
	provider := &Provider{
		OnLoaded:   OnLoaded,
		ConfigPath: ConfigPath,
	}
	err := provider.OnInit()
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (p *Provider) OnInit() error {
	p.Manager = viper.New()
	p.Manager.SetConfigType("yaml")
	configPath := "./"
	configName := "config"
	if p.ConfigPath != "" {
		configPath = filepath.Dir(p.ConfigPath)
		configName = filepath.Base(p.ConfigPath)
	}
	p.Manager.AddConfigPath(configPath)
	p.Manager.SetConfigName(configName)
	if _, err := os.Stat(filepath.Join(configPath, configName)); err == nil {
		err := p.Manager.ReadInConfig()
		if err != nil {
			return err
		}
	}
	if p.OnLoaded != nil {
		p.OnLoaded(p)
	}
	return nil
}
