package thumbnail

import (
	"context"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"mime"

	"github.com/allentom/harukap"
	"github.com/go-resty/resty/v2"
	"github.com/project-xpolaris/youplustoolkit/youlog"
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
	Prefix string
}

func (p *ThumbnailServicePlugin) Resize(ctx context.Context, input io.ReadCloser, option ThumbnailOption) (io.ReadCloser, error) {
	return p.Client.ResizeWithByte(ctx, input, option)
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
		prefix := "thumbnails."
		if p.Prefix != "" {
			prefix += p.Prefix
		}
		p.config = &ThumbnailServiceConfig{
			ServiceUrl: e.ConfigProvider.Manager.GetString(fmt.Sprintf("%s.url", prefix)),
			Enable:     e.ConfigProvider.Manager.GetBool(fmt.Sprintf("%s.enable", prefix)),
		}
	}
	logger.WithFields(map[string]interface{}{
		"enable": p.config.Enable,
		"url":    p.config.ServiceUrl,
	}).Info("thumbnail service config")
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

func (p *ThumbnailServicePlugin) GetPluginConfig() map[string]interface{} {
	if p.config == nil {
		return nil
	}
	return map[string]interface{}{
		"enable": p.config.Enable,
		"url":    p.config.ServiceUrl,
	}
}

type ThumbnailClient struct {
	BaseUrl string
}

func NewThumbnailClient(baseUrl string) *ThumbnailClient {
	return &ThumbnailClient{
		BaseUrl: baseUrl,
	}
}

func (c *ThumbnailClient) ResizeWithByte(ctx context.Context, input io.ReadCloser, option ThumbnailOption) (io.ReadCloser, error) {
	_, format, err := image.DecodeConfig(input)
	if err != nil {
		return nil, err
	}
	filename := "file" + mime.TypeByExtension("."+format)

	req := resty.New().R().
		SetFileReader("file", filename, input).
		SetContext(ctx)
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
	if err != nil {
		return nil, err
	}
	return response.RawBody(), nil
}
func (o *ThumbnailOption) GetSize(imageWidth, imageHeight int) (thumbnailWidth int, thumbnailHeight int) {
	switch o.Mode {
	case "width":
		thumbnailWidth = o.MaxWidth
		thumbnailHeight = int(float64(o.MaxWidth) * float64(imageHeight) / float64(imageWidth))
	case "height":
		thumbnailHeight = o.MaxHeight
		thumbnailWidth = int(float64(o.MaxHeight) * float64(imageWidth) / float64(imageHeight))
	case "resize":
		thumbnailWidth = o.MaxWidth
		thumbnailHeight = o.MaxHeight
	default:
		widthRatio := float64(imageWidth) / float64(o.MaxWidth)
		heightRatio := float64(imageHeight) / float64(o.MaxHeight)
		if widthRatio > heightRatio {
			thumbnailWidth = o.MaxWidth
			thumbnailHeight = int(float64(o.MaxWidth) * float64(imageHeight) / float64(imageWidth))
		} else {
			thumbnailHeight = o.MaxHeight
			thumbnailWidth = int(float64(o.MaxHeight) * float64(imageWidth) / float64(imageHeight))
		}
	}
	return
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
	if err != nil {
		return err
	}
	thumbnailContent := response.Body()
	err = ioutil.WriteFile(output, thumbnailContent, 0644)
	if err != nil {
		return err
	}
	return err
}
func (c *ThumbnailClient) GenerateAsRaw(sourcePath string, output string, option ThumbnailOption) (io.ReadCloser, error) {
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
	if err != nil {
		return nil, err
	}
	thumbnailContent := response.RawBody()
	return thumbnailContent, nil
}
func (c *ThumbnailClient) Resize(sourcePath string, option ThumbnailOption) ([]byte, error) {
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
	if err != nil {
		return nil, err
	}
	return thumbnailContent, nil
}
func (c *ThumbnailClient) Check() error {
	_, err := resty.New().R().Get(c.BaseUrl + "/info")
	if err != nil {
		return err
	}
	return nil
}
