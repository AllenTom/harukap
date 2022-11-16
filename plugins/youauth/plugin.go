package youauth

import (
	"github.com/allentom/haruka"
	"github.com/allentom/harukap"
	"github.com/allentom/harukap/commons"
	"github.com/project-xpolaris/youplustoolkit/youlink"
	"net/http"
	"net/url"
)

type OauthPlugin struct {
	Client          *YouAuthClient
	ConfigPrefix    string
	AuthFromToken   func(token string) (commons.AuthUser, error)
	OauthUrl        string
	PasswordAuthUrl string
}

func (p *OauthPlugin) getConfig(name string) string {
	if p.ConfigPrefix == "" {
		return name
	}
	return p.ConfigPrefix + "." + name
}
func (p *OauthPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	if p.ConfigPrefix == "" {
		p.ConfigPrefix = "auth"
	}
	p.Client = &YouAuthClient{}
	p.Client.BaseUrl = e.ConfigProvider.Manager.GetString(p.getConfig("url"))
	p.Client.AppId = e.ConfigProvider.Manager.GetString(p.getConfig("appid"))
	p.Client.Secret = e.ConfigProvider.Manager.GetString(p.getConfig("secret"))
	p.Client.Init()
	return nil
}
func (p *OauthPlugin) GetOauthPlugin() *OauthAuthPlugin {
	return &OauthAuthPlugin{
		OauthPlugin: p,
	}
}
func (p *OauthPlugin) GetPasswordPlugin() *PasswordAuthPlugin {
	return &PasswordAuthPlugin{
		OauthPlugin: p,
	}
}

func (p *OauthPlugin) GetOauthHandler(onAuth func(code string) (accessToken string, username string, err error)) haruka.RequestHandler {
	return func(context *haruka.Context) {
		code := context.GetQueryString("code")
		accessToken, username, err := onAuth(code)
		if err != nil {
			youlink.AbortErrorWithStatus(err, context, http.StatusInternalServerError)
			return
		}
		context.JSON(haruka.JSON{
			"success": true,
			"data": haruka.JSON{
				"accessToken": accessToken,
				"username":    username,
			},
		})
	}
}

type OauthAuthPlugin struct {
	*OauthPlugin
}

func (p *OauthAuthPlugin) AuthName() string {
	return "youauth"
}
func (p *OauthAuthPlugin) TokenTypeName() string {
	return "youauth"
}
func (p *OauthAuthPlugin) GetAuthUserByToken(token string) (commons.AuthUser, error) {
	if p.AuthFromToken == nil {
		return nil, nil
	}
	return p.AuthFromToken(token)
}
func (p *OauthAuthPlugin) GetAuthInfo() (*commons.AuthInfo, error) {
	authUrl, err := p.GetOauthUrl()
	if err != nil {
		return nil, err
	}
	authInfo := &commons.AuthInfo{
		Name: "YouAuth",
		Type: commons.AuthTypeWebOauth,
		Url:  authUrl,
	}
	return authInfo, nil
}
func (p *OauthAuthPlugin) GetOauthUrl() (string, error) {
	oauthUrl, err := url.Parse(p.Client.BaseUrl)
	if err != nil {
		return "", err
	}
	oauthUrl.Path = "/login"
	q := oauthUrl.Query()
	q.Add("client_id", p.Client.AppId)
	oauthUrl.RawQuery = q.Encode()
	return oauthUrl.String(), nil
}

type PasswordAuthPlugin struct {
	*OauthPlugin
}

func (p *PasswordAuthPlugin) GetAuthInfo() (*commons.AuthInfo, error) {
	authInfo := &commons.AuthInfo{
		Name: "YouAuthWithPassword",
		Type: commons.AuthTypeBase,
		Url:  p.PasswordAuthUrl,
	}
	return authInfo, nil
}

func (p *PasswordAuthPlugin) AuthName() string {
	return "youauthwithpassword"
}

func (p *PasswordAuthPlugin) GetAuthUserByToken(token string) (commons.AuthUser, error) {
	if p.AuthFromToken == nil {
		return nil, nil
	}
	return p.AuthFromToken(token)
}

func (p *PasswordAuthPlugin) TokenTypeName() string {
	return "youauth"
}
