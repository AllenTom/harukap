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
	Client       *YouAuthClient
	ConfigPrefix string
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
func (p *OauthPlugin) GetAuthInfo() (*commons.AuthInfo, error) {
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

func (p *OauthPlugin) GetOauthUrl() (string, error) {
	oauthUrl, err := url.Parse(p.Client.BaseUrl)
	if err != nil {
		return "", err
	}
	oauthUrl.Path = "/login"
	q := oauthUrl.Query()
	q.Add("appid", p.Client.AppId)
	oauthUrl.RawQuery = q.Encode()
	return oauthUrl.String(), nil
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
