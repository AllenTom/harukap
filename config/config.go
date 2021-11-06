package config

import (
	"github.com/spf13/viper"
)

type Provider struct {
	Manager  *viper.Viper
	OnLoaded func(provider *Provider)
}

func NewProvider(OnLoaded func(provider *Provider)) (*Provider, error) {
	provider := &Provider{
		OnLoaded: OnLoaded,
	}
	err := provider.OnInit()
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (p *Provider) OnInit() error {
	p.Manager = viper.New()
	p.Manager.AddConfigPath("./")
	p.Manager.AddConfigPath("../")
	p.Manager.SetConfigType("yaml")
	p.Manager.SetConfigName("config")
	err := p.Manager.ReadInConfig()
	if err != nil {
		return err
	}
	if p.OnLoaded != nil {
		p.OnLoaded(p)
	}
	return nil
}
