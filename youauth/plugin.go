package youauth

import (
	"github.com/allentom/haruka"
	"github.com/allentom/harukap"
	"github.com/project-xpolaris/youplustoolkit/youlink"
	"net/http"
	"net/url"
)

type OauthPlugin struct {
	Client *YouAuthClient
}

func (p *OauthPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	p.Client = &YouAuthClient{}
	p.Client.BaseUrl = e.ConfigProvider.Manager.GetString("auth.url")
	p.Client.AppId = e.ConfigProvider.Manager.GetString("auth.appid")
	p.Client.Secret = e.ConfigProvider.Manager.GetString("auth.secret")
	p.Client.Init()
	return nil
}

func (p *OauthPlugin) GetOauthUrl() (string, error) {
	oauthUrl, err := url.Parse(p.Client.BaseUrl)
	if err != nil {
		return "", err
	}
	oauthUrl.Path = "/login"
	q := oauthUrl.Query()
	q.Set("appid", p.Client.AppId)
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
