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
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"`
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

// LoginWithPassword uses YouAuth first-party /user/auth (no OAuth app required).
// Prefer this for system console / NAS gateway password login.
func (c *YouAuthClient) LoginWithPassword(username string, password string) (*GenerateTokenResponse, error) {
	type userAuthResponse struct {
		Success bool `json:"success"`
		Data    struct {
			Token    string `json:"token"`
			Username string `json:"username"`
			Id       int    `json:"id"`
		} `json:"data"`
		Err  string `json:"err"`
		Code string `json:"code"`
	}
	result := &userAuthResponse{}
	response, err := c.client.NewRequest().SetBody(map[string]string{
		"username": username,
		"password": password,
	}).SetResult(result).Post(c.BaseUrl + "/user/auth")
	if err != nil {
		return nil, err
	}
	body := response.Body()
	var baseBody map[string]interface{}
	if err = json.Unmarshal(body, &baseBody); err != nil {
		return nil, err
	}
	if err = c.processError(baseBody); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	if result.Data.Token == "" {
		return nil, errors.New("empty token from /user/auth")
	}
	return &GenerateTokenResponse{
		AccessToken: result.Data.Token,
		TokenType:   "Bearer",
	}, nil
}

// GrantWithPassword is the OAuth password grant (/token). client_id is optional.
// For first-party console login prefer LoginWithPassword.
func (c *YouAuthClient) GrantWithPassword(username string, password string) (*GenerateTokenResponse, error) {
	form := map[string]string{
		"username":   username,
		"password":   password,
		"grant_type": "password",
	}
	if c.AppId != "" {
		form["client_id"] = c.AppId
	}
	result := &GenerateTokenResponse{}
	response, err := c.client.NewRequest().SetFormData(form).SetResult(result).Post(c.BaseUrl + "/token")
	if err != nil {
		return nil, err
	}
	err = c.parseResponse(response, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
