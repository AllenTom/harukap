package config

import (
	"github.com/spf13/viper"
	"path/filepath"
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
	if p.ConfigPath != "" {
		p.Manager.AddConfigPath(filepath.Dir(p.ConfigPath))
		p.Manager.SetConfigName(filepath.Base(p.ConfigPath))
	} else {
		p.Manager.AddConfigPath("./")
		p.Manager.AddConfigPath("../")
		p.Manager.SetConfigName("config")
	}
	p.Manager.SetConfigType("yaml")
	err := p.Manager.ReadInConfig()
	if err != nil {
		return err
	}
	if p.OnLoaded != nil {
		p.OnLoaded(p)
	}
	return nil
}
