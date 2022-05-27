package handlers

import (
	"fmt"
	"github.com/allentom/haruka"
	"github.com/allentom/harukap"
	"github.com/spf13/viper"
)

func NewServicesInfoHandler(
	extend haruka.JSON,
	authPlugins []harukap.AuthPlugin,
	onError func(err error),
	configProvider *viper.Viper,
) haruka.RequestHandler {
	return func(context *haruka.Context) {
		authMaps := make([]interface{}, 0)
		configManager := configProvider
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
				for _, authPlugin := range authPlugins {
					if authPlugin.AuthName() == authType {
						authInfo, err := authPlugin.GetAuthInfo()
						if err != nil {
							onError(err)
							return
						}
						authMaps = append(authMaps, authInfo)
						break
					}
				}
			}
		}
		data := extend
		data["auth"] = authMaps
		context.JSON(data)
	}
}
