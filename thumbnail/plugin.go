package thumbnail

import (
	"fmt"
	"github.com/allentom/harukap"
	"github.com/go-resty/resty/v2"
	"github.com/project-xpolaris/youplustoolkit/youlog"
	"io/ioutil"
)

var DefaultThumbnailServicePlugin = &ThumbnailServicePlugin{}

type ThumbnailServiceConfig struct {
	Enable     bool
	ServiceUrl string
}
type ThumbnailServicePlugin struct {
	Client *ThumbnailClient
	config *ThumbnailServiceConfig
	Logger *youlog.Scope
}

func (p *ThumbnailServicePlugin) SetConfig(config *ThumbnailServiceConfig) {
	p.config = config
}
func (p *ThumbnailServicePlugin) OnInit(e *harukap.HarukaAppEngine) error {
	logger := e.LoggerPlugin.Logger.NewScope("ThumbnailPlugin")
	p.Logger = logger
	logger.Info("Init ThumbnailPlugin")
	if p.config == nil {
		logger.Info("Init ThumbnailPlugin with default config")
		p.config = &ThumbnailServiceConfig{
			ServiceUrl: e.ConfigProvider.Manager.GetString("thumbnails.service_url"),
			Enable:     e.ConfigProvider.Manager.GetBool("thumbnails.enable"),
		}
	}
	if !p.config.Enable {
		logger.Info("ThumbnailPlugin is disabled")
		return nil
	}
	logger.Info(fmt.Sprintf("connect to %s", p.config.ServiceUrl))
	p.Client = NewThumbnailClient(p.config.ServiceUrl)
	err := p.Client.Check()
	if err != nil {
		return err
	}
	logger.Info("Init ThumbnailPlugin success")
	return nil
}

type ThumbnailClient struct {
	BaseUrl string
}

func NewThumbnailClient(baseUrl string) *ThumbnailClient {
	return &ThumbnailClient{
		BaseUrl: baseUrl,
	}
}

type ThumbnailOption struct {
	MaxWidth  int    `hsource:"query" hname:"maxWidth"`
	MaxHeight int    `hsource:"query" hname:"maxHeight"`
	Mode      string `hsource:"query" hname:"mode"`
}

func (c *ThumbnailClient) Generate(sourcePath string, output string, option ThumbnailOption) error {
	req := resty.New().R().
		SetFile("file", sourcePath)
	if option.MaxWidth != 0 {
		req.SetQueryParam("maxWidth", fmt.Sprintf("%d", option.MaxWidth))
	}
	if option.MaxHeight != 0 {
		req.SetQueryParam("maxHeight", fmt.Sprintf("%d", option.MaxHeight))
	}
	if option.Mode != "" {
		req.SetQueryParam("mode", option.Mode)
	}
	response, err := req.Post(c.BaseUrl + "/generator")

	thumbnailContent := response.Body()
	err = ioutil.WriteFile(output, thumbnailContent, 0644)
	if err != nil {
		return err
	}
	return err
}

func (c *ThumbnailClient) Check() error {
	_, err := resty.New().R().Get(c.BaseUrl + "/info")
	if err != nil {
		return err
	}
	return nil
}
