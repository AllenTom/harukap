package auth

import (
	"fmt"
	"github.com/allentom/haruka"
	"github.com/allentom/harukap"
	"github.com/allentom/harukap/config"
)

type AuthModuleConfig struct {
	EnableAnonymous bool
}
type AuthModule struct {
	Plugins        []harukap.AuthPlugin
	AuthMiddleware AuthMiddleware
	ConfigProvider *config.Provider
	NoAuthPath     []string
	Config         AuthModuleConfig
}

func (m *AuthModule) InitModule() {
	authConfig := AuthModuleConfig{}
	configer := m.ConfigProvider.Manager
	for key, _ := range configer.GetStringMap("auth") {
		configType := configer.GetString(fmt.Sprintf("auth.%s.type", key))
		enable := configer.GetBool(fmt.Sprintf("auth.%s.enable", key))
		if configType == "anonymous" && enable {
			authConfig.EnableAnonymous = true
		}
	}
	if m.NoAuthPath == nil {
		m.NoAuthPath = []string{}
	}
	m.Config = authConfig
	m.AuthMiddleware = AuthMiddleware{
		Module: m,
	}
}

func (m *AuthModule) GetAuthPluginByName(name string) harukap.AuthPlugin {
	for _, plugin := range m.Plugins {
		if plugin.TokenTypeName() == name {
			return plugin
		}
	}
	return nil
}

func (m *AuthModule) GetAuthConfig() ([]interface{}, error) {
	authMaps := make([]interface{}, 0)
	configManager := m.ConfigProvider.Manager
	for key := range configManager.GetStringMap("auth") {
		authType := configManager.GetString(fmt.Sprintf("auth.%s.type", key))
		enable := configManager.GetBool(fmt.Sprintf("auth.%s.enable", key))
		if !enable {
			continue
		}
		if authType == "anonymous" {
			authMaps = append(authMaps, haruka.JSON{
				"type": "anonymous",
				"name": "Anonymous",
				"url":  "",
			})
		} else {
			for _, authPlugin := range m.Plugins {
				if authPlugin.AuthName() == authType {
					authInfo, err := authPlugin.GetAuthInfo()
					if err != nil {
						return nil, err
					}
					authMaps = append(authMaps, authInfo)
					break
				}
			}
		}
	}
	return authMaps, nil
}
