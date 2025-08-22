package harukap

import "github.com/allentom/harukap/commons"

type HarukaPlugin interface {
	OnInit(e *HarukaAppEngine) error
}

// PluginWithConfig 可选接口：插件实现后即可提供其配置快照
type PluginWithConfig interface {
	GetPluginConfig() map[string]interface{}
}

type AuthPlugin interface {
	GetAuthInfo() (*commons.AuthInfo, error)
	AuthName() string
	GetAuthUserByToken(token string) (commons.AuthUser, error)
	TokenTypeName() string
}
