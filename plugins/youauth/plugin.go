package youauth

import (
	"fmt"

	"github.com/allentom/harukap"
	"github.com/allentom/harukap/commons"
	"github.com/allentom/harukap/plugins/nacos"
)

// YouAuthPlugin 把服务接到 YouAuth：令牌校验（AuthFromToken）+ 密码登录入口描述（PasswordAuthUrl）。
// YouAuth 已不再支持 OAuth 应用与授权码登录，这里只有第一方密码登录。
type YouAuthPlugin struct {
	Client          *YouAuthClient
	ConfigPrefix    string
	AuthFromToken   func(token string) (commons.AuthUser, error)
	PasswordAuthUrl string
}

func (p *YouAuthPlugin) getConfig(name string) string {
	if p.ConfigPrefix == "" {
		return name
	}
	return p.ConfigPrefix + "." + name
}
func (p *YouAuthPlugin) OnInit(e *harukap.HarukaAppEngine) error {
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
	// youlog 配置输出
	logger := e.LoggerPlugin.Logger.NewScope("YouAuthPlugin")
	logger.WithFields(map[string]interface{}{
		"baseUrl": p.Client.BaseUrl,
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

func (p *YouAuthPlugin) GetAuthPlugin() *PasswordAuthPlugin {
	return &PasswordAuthPlugin{
		YouAuthPlugin: p,
	}
}

type PasswordAuthPlugin struct {
	*YouAuthPlugin
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

func (p *YouAuthPlugin) GetPluginConfig() map[string]interface{} {
	cfg := map[string]interface{}{}
	if p.Client != nil {
		cfg["baseUrl"] = p.Client.BaseUrl
	}
	cfg["prefix"] = p.ConfigPrefix
	return cfg
}
