package tagger

import (
	"github.com/allentom/harukap"
)

type ImageTaggerPlugin struct {
	Client *Client
	Enable bool
}

func (i *ImageTaggerPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	logger := e.LoggerPlugin.Logger.NewScope("ImageTaggerPlugin")

	isEnable := e.ConfigProvider.Manager.GetBool("imagetagger.enable")
	i.Enable = isEnable
	if !isEnable {
		logger.Info("ImageTaggerPlugin is disabled")
		return nil
	}

	logger.Info("Init ImageTaggerPlugin")
	baseUrl := e.ConfigProvider.Manager.GetString("imagetagger.url")
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

func NewImageTaggerPlugin() *ImageTaggerPlugin {
	return &ImageTaggerPlugin{}
}
