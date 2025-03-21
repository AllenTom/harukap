package tagger

import (
	"bytes"
	"errors"
	"fmt"
	util "github.com/allentom/harukap/utils"
	"github.com/go-resty/resty/v2"
	"io"
)

type Client struct {
	BaseUrl string
	client  *resty.Client
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
	client := resty.New()
	return &Client{
		BaseUrl: baseUrl,
		client:  client,
	}
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
	result := &ResponseWrap[[]ImageTag]{}
	_, err = c.client.R().
		SetFileReader("file", fmt.Sprintf("image.%s", imageFormat), bytes.NewBuffer(rawImage)).
		SetResult(result).
		SetQueryParam("model", taggerModel).
		SetQueryParam("threshold", fmt.Sprintf("%f", threshold)).
		Post(c.BaseUrl + "/tagimage")
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, errors.New(fmt.Sprintf("tagger failed: %v", result.Error))
	}
	return result.Data, nil
}

func (c *Client) GetInfo() (*ServiceInfo, error) {
	result := &ServiceInfo{}
	_, err := c.client.R().
		SetResult(result).
		Get(c.BaseUrl + "/info")
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, errors.New(fmt.Sprintf("get info failed: %v", result))
	}
	return result, nil
}

func (c *Client) SwitchModel(name string) error {
	result := &ResponseWrap[struct {
		Result bool `json:"result"`
	}]{}
	body := map[string]string{
		"model": name,
	}
	_, err := c.client.R().
		SetBody(body).
		SetResult(result).
		Post(c.BaseUrl + "/switch")
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
	result := &ResponseWrap[TaggerState]{}
	_, err := c.client.R().
		SetResult(result).
		Get(c.BaseUrl + "/state")
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, errors.New(fmt.Sprintf("get info failed: %v", result))
	}
	return &result.Data, nil
}
