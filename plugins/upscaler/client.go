package upscaler

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

// Models:
//RealESRGAN_x4plus
//RealESRNet_x4plus
//RealESRGAN_x4plus_anime_6B
//RealESRGAN_x2plus
//realesr-animevideov3
//realesr-general-x4v3

var ModelRealESRGANX4plus = "RealESRGAN_x4plus"
var ModelRealESRNetX4plus = "RealESRNet_x4plus"
var ModelRealESRGANX4plusAnime6B = "RealESRGAN_x4plus_anime_6B"
var ModelRealESRGANX2plus = "RealESRGAN_x2plus"
var ModelRealSRAnimeVideoV3 = "realesr-animevideov3"
var ModelRealSRGeneralX4V3 = "realesr-general-x4v3"

type UpscaleOptions struct {
	Model       string  `json:"model_name"`
	FaceEnhance bool    `json:"face_enhance"`
	OutScale    float64 `json:"out_scale"`
}

func (c *Client) Upscale(reader io.Reader, option *UpscaleOptions) ([]byte, error) {
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
	req := c.client.R().
		SetFileReader("file", fmt.Sprintf("image.%s", imageFormat), bytes.NewBuffer(rawImage)).
		SetQueryParam("model", option.Model)
	if option.OutScale != 0 {
		req.SetQueryParam("out_scale", fmt.Sprintf("%f", option.OutScale))
	}
	if option.FaceEnhance {
		req.SetQueryParam("face_enhance", "1")
	}
	resp, err := req.
		Post(c.BaseUrl + "/upscale")
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
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
