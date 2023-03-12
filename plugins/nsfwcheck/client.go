package nsfwcheck

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

func NewClient(baseUrl string) *Client {
	client := resty.New()
	return &Client{
		BaseUrl: baseUrl,
		client:  client,
	}
}

func (c *Client) Predict(reader io.Reader) ([]Predictions, error) {
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
	result := &BaseResponse[[]Predictions]{}
	_, err = c.client.R().
		SetFileReader("image", fmt.Sprintf("image.%s", imageFormat), bytes.NewBuffer(rawImage)).
		SetResult(result).
		Post(c.BaseUrl + "/predict")
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, errors.New(fmt.Sprintf("predict failed: %v", result.Err))
	}
	return result.Data, nil
}
func (c *Client) Info() (*InfoResponse, error) {
	result := &InfoResponse{}
	_, err := c.client.R().
		SetResult(result).
		Get(c.BaseUrl + "/info")
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, errors.New(fmt.Sprintf("get info failed: %v", result.Err))
	}
	return result, nil
}

type BaseResponse[T any] struct {
	Data    T      `json:"data"`
	Success bool   `json:"success"`
	Err     string `json:"err"`
}
type Predictions struct {
	Classname   string  `json:"classname"`
	Probability float64 `json:"probability"`
}

type InfoResponse struct {
	Success bool   `json:"success"`
	Err     string `json:"err"`
	Name    string `json:"name"`
}
