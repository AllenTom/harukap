package deepdanbooru

import (
	"bytes"
	"errors"
	"fmt"
	util "github.com/allentom/harukap/utils"
	"github.com/go-resty/resty/v2"
	"io"
	"time"
)

type Client struct {
	client *resty.Client
	conf   *Config
}

type Config struct {
	Url string
}

func NewClient(conf *Config) *Client {
	client := resty.New()
	return &Client{
		client: client,
		conf:   conf,
	}
}
func (c *Client) SetTimeout(timeout time.Duration) {
	c.client.SetTimeout(timeout)
}
func (c *Client) Tagging(reader io.Reader) ([]Predictions, error) {
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
		SetFileReader("file", fmt.Sprintf("image.%s", imageFormat), bytes.NewBuffer(rawImage)).
		SetResult(result).
		Post(c.conf.Url + "/tagging")
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
		Get(c.conf.Url + "/info")
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
	Tag  string  `json:"tag"`
	Prob float64 `json:"prob"`
}

type InfoResponse struct {
	Success bool   `json:"success"`
	Err     string `json:"err"`
	Name    string `json:"name"`
}
