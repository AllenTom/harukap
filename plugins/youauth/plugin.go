package youauth

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/allentom/haruka"
	"github.com/allentom/harukap"
	"github.com/allentom/harukap/commons"
	"github.com/allentom/harukap/plugins/nacos"
	util "github.com/allentom/harukap/utils"
	"github.com/project-xpolaris/youplustoolkit/youlink"
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
	// 自动定位到 auth.<providerKey>（如 auth.youauth），便于将 nacos 配置放在 youauth 节点下
	configer := e.ConfigProvider.Manager
	if p.ConfigPrefix == "auth" {
		for key := range configer.GetStringMap("auth") {
			if configer.GetString(fmt.Sprintf("auth.%s.type", key)) == "youauth" {
				p.ConfigPrefix = fmt.Sprintf("auth.%s", key)
				break
			}
		}
	}
	p.Client = &YouAuthClient{}
	// 优先尝试通过 Nacos 发现 YouAuth 地址
	useNacos := configer.GetBool(p.getConfig("nacos.enable"))
	if useNacos {
		serviceName := configer.GetString(p.getConfig("nacos.serviceName"))
		if serviceName == "" {
			serviceName = "youauth"
		}
		group := configer.GetString(p.getConfig("nacos.group"))
		if group == "" {
			group = "DEFAULT_GROUP"
		}
		scheme := configer.GetString(p.getConfig("nacos.scheme"))
		if scheme == "" {
			scheme = "http"
		}
		for _, pl := range e.Plugins {
			if np, ok := pl.(*nacos.NacosPlugin); ok && np != nil {
				inst, err := np.GetServiceInstance(serviceName, group)
				if err == nil && inst != nil && inst.Ip != "" && inst.Port > 0 {
					p.Client.BaseUrl = fmt.Sprintf("%s://%s:%d", scheme, inst.Ip, inst.Port)
					break
				}
			}
		}
	}
	// 回退到静态配置
	if p.Client.BaseUrl == "" {
		p.Client.BaseUrl = configer.GetString(p.getConfig("url"))
	}
	p.Client.AppId = configer.GetString(p.getConfig("appid"))
	p.Client.Secret = configer.GetString(p.getConfig("secret"))
	// youlog 配置输出
	logger := e.LoggerPlugin.Logger.NewScope("YouAuthPlugin")
	logger.WithFields(map[string]interface{}{
		"baseUrl": p.Client.BaseUrl,
		"appid":   p.Client.AppId,
		"secret":  util.MaskKeepHeadTail(p.Client.Secret, 2, 2),
		"nacos": map[string]interface{}{
			"enable": useNacos,
			"serviceName": func() string {
				v := configer.GetString(p.getConfig("nacos.serviceName"))
				if v == "" {
					return "youauth"
				}
				return v
			}(),
			"group": func() string {
				v := configer.GetString(p.getConfig("nacos.group"))
				if v == "" {
					return "DEFAULT_GROUP"
				}
				return v
			}(),
			"scheme": func() string {
				v := configer.GetString(p.getConfig("nacos.scheme"))
				if v == "" {
					return "http"
				}
				return v
			}(),
		},
	}).Info("youauth config")
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

func (p *OauthPlugin) GetPluginConfig() map[string]interface{} {
	cfg := map[string]interface{}{}
	if p.Client != nil {
		cfg["baseUrl"] = p.Client.BaseUrl
		cfg["appid"] = p.Client.AppId
		cfg["secret"] = util.MaskKeepHeadTail(p.Client.Secret, 2, 2)
	}
	cfg["prefix"] = p.ConfigPrefix
	return cfg
}
