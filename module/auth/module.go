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
	Config         AuthModuleConfig
	CacheStore     *TokenStoreManager
}

func (m *AuthModule) AddCacheStore(convert Serializer) {
	m.CacheStore = &TokenStoreManager{
		Serializer: convert,
		module:     m,
	}
}
func (m *AuthModule) InitModule() error {
	authConfig := AuthModuleConfig{}
	configer := m.ConfigProvider.Manager
	for key := range configer.GetStringMap("auth") {
		configType := configer.GetString(fmt.Sprintf("auth.%s.type", key))
		enable := configer.GetBool(fmt.Sprintf("auth.%s.enable", key))
		if configType == "anonymous" && enable {
			authConfig.EnableAnonymous = true
		}
	}
	m.Config = authConfig
	m.AuthMiddleware = AuthMiddleware{
		Module: m,
	}
	if m.CacheStore != nil {
		err := m.CacheStore.Init()
		if err != nil {
			return err
		}
	}
	return nil
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
