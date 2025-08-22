package upscaler

import (
	"github.com/allentom/harukap"
)

type ImageUpscalerPlugin struct {
	Client *Client
	Enable bool
}

func (i *ImageUpscalerPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	logger := e.LoggerPlugin.Logger.NewScope("ImageUpscalerPlugin")

	isEnable := e.ConfigProvider.Manager.GetBool("imageupscaler.enable")
	i.Enable = isEnable
	if !isEnable {
		logger.Info("ImageUpscalerPlugin is disabled")
		return nil
	}

	logger.Info("Init ImageUpscalerPlugin")
	baseUrl := e.ConfigProvider.Manager.GetString("imageupscaler.url")
	logger.WithFields(map[string]interface{}{
		"enable":  isEnable,
		"baseUrl": baseUrl,
	}).Info("imageupscaler config")
	i.Client = NewClient(baseUrl)
	logger.Info("check connection")
	info, err := i.Client.GetInfo()
	if err != nil {
		i.Client = nil
		logger.Error(err)
		return nil
	}
	if !info.Success {
		i.Client = nil
		logger.Error("connection failed")
		return nil
	}
	logger.Info("connection success")
	return nil
}

func (i *ImageUpscalerPlugin) IsEnable() bool {
	return i.Enable && i.Client != nil
}

func NewImageUpscalerPlugin() *ImageUpscalerPlugin {
	return &ImageUpscalerPlugin{}
}

func (i *ImageUpscalerPlugin) GetPluginConfig() map[string]interface{} {
	base := ""
	if i.Client != nil {
		base = i.Client.BaseUrl
	}
	return map[string]interface{}{
		"enable":  i.Enable,
		"baseUrl": base,
	}
}
