package tagger

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	util "github.com/allentom/harukap/utils"
	"github.com/go-resty/resty/v2"
)

// ClientConfig 定义了客户端的配置选项
type ClientConfig struct {
	// HTTP 客户端配置
	Timeout          time.Duration // 请求超时时间
	RetryCount       int           // 重试次数
	RetryWaitTime    time.Duration // 重试等待时间
	MaxRetryWaitTime time.Duration // 最大重试等待时间

	// 基础 URL（直接连接模式）
	BaseURL string

	// 其他配置
	EnableDebug bool // 是否启用调试模式
}

// DefaultConfig 返回默认配置
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:          30 * time.Second,
		RetryCount:       3,
		RetryWaitTime:    1 * time.Second,
		MaxRetryWaitTime: 10 * time.Second,
		EnableDebug:      false,
	}
}

type Client struct {
	BaseUrl string
	client  *resty.Client
	config  *ClientConfig
}

type ResponseWrap[T interface{}] struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Data    T      `json:"data"`
}

type ImageTag struct {
	Tag  string  `json:"tag"`
	Rank float64 `json:"rank"`
}

type ServiceInfo struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
}

func NewClient(baseUrl string) *Client {
	config := DefaultConfig()
	config.BaseURL = baseUrl
	return NewClientWithConfig(config)
}

// NewClientWithConfig 使用自定义配置创建客户端
func NewClientWithConfig(config *ClientConfig) *Client {
	client := resty.New()

	// 配置 HTTP 客户端
	client.SetTimeout(config.Timeout)
	client.SetRetryCount(config.RetryCount)
	client.SetRetryWaitTime(config.RetryWaitTime)
	client.SetRetryMaxWaitTime(config.MaxRetryWaitTime)

	// 配置调试模式
	if config.EnableDebug {
		client.SetDebug(true)
	}

	c := &Client{
		BaseUrl: config.BaseURL,
		client:  client,
		config:  config,
	}

	return c
}

// normalizeURL 确保 URL 不包含尾部斜杠
func normalizeURL(url string) string {
	if url == "" {
		return url
	}
	// 移除尾部斜杠
	for len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	return url
}

func (c *Client) getServiceUrl() (string, error) {
	return normalizeURL(c.BaseUrl), nil
}

//wd14_MOAT
//wd14_SwinV2
//wd14_ConvNext
//wd14_ConvNextV2
//wd14_ViT
//DeepDanbooru
//clip2

var ModelWd14MOAT = "wd14_MOAT"
var ModelWd14SwinV2 = "wd14_SwinV2"
var ModelWd14ConvNext = "wd14_ConvNext"
var ModelWd14ConvNextV2 = "wd14_ConvNextV2"
var ModelViT = "wd14_ViT"
var ModelDeepDanbooru = "DeepDanbooru"
var ModelClip2 = "clip2"
var ModelAuto = "auto"

func (c *Client) TagImage(reader io.Reader, taggerModel string, threshold float64) ([]ImageTag, error) {
	rawImage, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	imageFormat, err := util.GuessImageFormat(bytes.NewBuffer(rawImage))
	if err != nil {
		return nil, err
	}
	if imageFormat == "" {
		return nil, errors.New("invalid image format")
	}

	// 获取服务地址
	serviceUrl, err := c.getServiceUrl()
	if err != nil {
		return nil, fmt.Errorf("failed to get service url: %v", err)
	}

	result := &ResponseWrap[[]ImageTag]{}
	_, err = c.client.R().
		SetFileReader("file", fmt.Sprintf("image.%s", imageFormat), bytes.NewBuffer(rawImage)).
		SetResult(result).
		SetQueryParam("model", taggerModel).
		SetQueryParam("threshold", fmt.Sprintf("%f", threshold)).
		Post(fmt.Sprintf("%s/tagimage", serviceUrl))
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	if !result.Success {
		return nil, errors.New(fmt.Sprintf("tagger failed: %v", result.Error))
	}
	return result.Data, nil
}

func (c *Client) GetInfo() (*ServiceInfo, error) {
	// 获取服务地址
	serviceUrl, err := c.getServiceUrl()
	if err != nil {
		return nil, err
	}

	result := &ServiceInfo{}
	_, err = c.client.R().
		SetResult(result).
		Get(fmt.Sprintf("%s/info", serviceUrl))
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, errors.New(fmt.Sprintf("get info failed: %v", result))
	}
	return result, nil
}

func (c *Client) SwitchModel(name string) error {
	// 获取服务地址
	serviceUrl, err := c.getServiceUrl()
	if err != nil {
		return err
	}

	result := &ResponseWrap[struct {
		Result bool `json:"result"`
	}]{}
	body := map[string]string{
		"model": name,
	}
	_, err = c.client.R().
		SetBody(body).
		SetResult(result).
		Post(fmt.Sprintf("%s/switch", serviceUrl))
	if err != nil {
		return err
	}
	if !result.Success {
		return errors.New(fmt.Sprintf("switch model failed: %v", result.Error))
	}
	return nil
}

type TaggerState struct {
	ModelName string   `json:"modelName"`
	ModelList []string `json:"modelList"`
}

func (c *Client) GetTaggerState() (*TaggerState, error) {
	// 获取服务地址
	serviceUrl, err := c.getServiceUrl()
	if err != nil {
		return nil, err
	}

	result := &ResponseWrap[TaggerState]{}
	_, err = c.client.R().
		SetResult(result).
		Get(fmt.Sprintf("%s/state", serviceUrl))
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, errors.New(fmt.Sprintf("get info failed: %v", result))
	}
	return &result.Data, nil
}
