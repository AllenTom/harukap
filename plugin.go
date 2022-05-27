package harukap

import "github.com/allentom/harukap/commons"

type HarukaPlugin interface {
	OnInit(e *HarukaAppEngine) error
}

type AuthPlugin interface {
	GetAuthInfo() (*commons.AuthInfo, error)
	AuthName() string
	GetAuthUserByToken(token string) (commons.AuthUser, error)
	TokenTypeName() string
}
