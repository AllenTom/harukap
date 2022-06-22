package youauth

import (
	"encoding/json"
	"errors"
	"github.com/go-resty/resty/v2"
)

var (
	TokenExpiredError = errors.New("token expired")
)

type YouAuthResponse interface {
	GetSuccess() bool
	GetError() string
	GetCode() string
}
type BaseResponse struct {
	Success bool   `json:"success"`
	Err     string `json:"err"`
	Code    string `json:"code"`
}

func (r *BaseResponse) GetSuccess() bool {
	return r.Success
}

func (r *BaseResponse) GetError() string {
	return r.Err
}

func (r *BaseResponse) GetCode() string {
	return r.Code
}

type YouAuthClient struct {
	BaseUrl string
	AppId   string
	Secret  string
	client  *resty.Client
}

func (c *YouAuthClient) Init() {
	if c.client == nil {
		c.client = resty.New()
	}
	return
}

type GenerateTokenResponse struct {
	Data TokenData `json:"data"`
}
type TokenData struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (c *YouAuthClient) processError(response YouAuthResponse) error {
	if response.GetSuccess() {
		return nil
	}
	switch response.GetCode() {
	case "1001":
		return TokenExpiredError
	default:
		return errors.New(response.GetError())
	}
}
func (c *YouAuthClient) GetAccessToken(authCode string) (*TokenData, error) {
	result := &GenerateTokenResponse{}
	response, err := c.client.NewRequest().SetBody(map[string]interface{}{
		"appId":  c.AppId,
		"secret": c.Secret,
		"code":   authCode,
	}).SetResult(result).Post(c.BaseUrl + "/oauth/token")
	if err != nil {
		return nil, err
	}
	err = c.parseResponse(response, result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

type GetCurrentUserResponse struct {
	Data UserData `json:"data"`
}
type UserData struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
}

func (c *YouAuthClient) parseResponse(response *resty.Response, data interface{}) error {
	body := response.Body()
	var baseBody BaseResponse
	err := json.Unmarshal(body, &baseBody)
	if err != nil {
		return err
	}
	err = c.processError(&baseBody)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, data)
	return err
}
func (c *YouAuthClient) GetCurrentUser(accessToken string) (*UserData, error) {
	result := &GetCurrentUserResponse{}
	response, err := c.client.NewRequest().SetQueryParam("token", accessToken).SetResult(result).Get(c.BaseUrl + "/auth/current")
	if err != nil {
		return nil, err
	}
	err = c.parseResponse(response, result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *YouAuthClient) RefreshAccessToken(refreshToken string) (*TokenData, error) {
	result := &GenerateTokenResponse{}
	response, err := c.client.NewRequest().SetBody(map[string]interface{}{
		"secret":       c.Secret,
		"refreshToken": refreshToken,
	}).SetResult(result).Post(c.BaseUrl + "/oauth/refresh")
	if err != nil {
		return nil, err
	}
	err = c.parseResponse(response, result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}
