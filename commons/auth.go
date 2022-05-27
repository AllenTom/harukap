package commons

const (
	AuthTypeWebOauth = "weboauth"
	AuthTypeBase     = "base"
)
const (
	AuthProviderYouAuth = "youauth"
	AuthProviderYouPlus = "YouPlusService"
)

type AuthInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Url  string `json:"url"`
}

type AuthUser interface {
}
