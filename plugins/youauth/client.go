package youauth

import (
	"encoding/json"
	"errors"
	"github.com/go-resty/resty/v2"
)

var (
	TokenExpiredError = errors.New("token expired")
)

type BaseResponse struct {
	Success bool   `json:"success"`
	Err     string `json:"err"`
	Code    string `json:"code"`
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
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Expire       int64  `json:"expire_in"`
	TokenType    string `json:"token_type"`
}

func (c *YouAuthClient) processError(response map[string]interface{}) error {
	if ok, okOk := response["success"].(bool); okOk {
		if ok {
			return nil
		}
	}
	if errcode, errOk := response["code"].(string); errOk {
		switch errcode {
		case "1001":
			return TokenExpiredError
		default:
			return errors.New(response["err"].(string))
		}
	}
	return nil
}
func (c *YouAuthClient) GetAccessToken(authCode string) (*GenerateTokenResponse, error) {
	result := &GenerateTokenResponse{}
	response, err := c.client.NewRequest().SetFormData(map[string]string{
		"code":       authCode,
		"grant_type": "authorization_code",
	}).SetResult(result).Post(c.BaseUrl + "/token")
	if err != nil {
		return nil, err
	}
	err = c.parseResponse(response, result)
	if err != nil {
		return nil, err
	}
	return result, nil
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
	var baseBody map[string]interface{}
	err := json.Unmarshal(body, &baseBody)
	if err != nil {
		return err
	}
	err = c.processError(baseBody)
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

func (c *YouAuthClient) RefreshAccessToken(refreshToken string) (*GenerateTokenResponse, error) {
	result := &GenerateTokenResponse{}
	response, err := c.client.NewRequest().SetFormData(map[string]string{
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
	}).SetResult(result).Post(c.BaseUrl + "/token")
	if err != nil {
		return nil, err
	}
	err = c.parseResponse(response, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *YouAuthClient) GrantWithPassword(username string, password string) (*GenerateTokenResponse, error) {
	result := &GenerateTokenResponse{}
	response, err := c.client.NewRequest().SetFormData(map[string]string{
		"username":   username,
		"password":   password,
		"grant_type": "password",
		"client_id":  c.AppId,
	}).SetResult(result).Post(c.BaseUrl + "/token")
	if err != nil {
		return nil, err
	}
	err = c.parseResponse(response, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
