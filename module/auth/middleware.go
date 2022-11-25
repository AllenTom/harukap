package auth

import (
	"github.com/allentom/haruka"
	"github.com/allentom/harukap/commons"
)

type AuthMiddleware struct {
	OnError       func(c *haruka.Context, err error)
	Module        *AuthModule
	RequestFilter func(c *haruka.Context) bool
}

func (m AuthMiddleware) OnRequest(c *haruka.Context) {
	var err error
	if m.RequestFilter != nil && !m.RequestFilter(c) {
		return
	}
	jwtToken := m.Module.ParseAuthHeader(c)
	if m.Module.Config.EnableAnonymous && len(jwtToken) == 0 {
		return
	}
	var claims commons.AuthUser
	if m.Module.CacheStore != nil {
		claims, err = m.Module.CacheStore.GetUserByToken(jwtToken)
		if err != nil {
			m.OnError(c, err)
			return
		}
	} else {
		claims, err = m.Module.ParseToken(jwtToken)
		if err != nil {
			m.OnError(c, err)
			return
		}
	}
	c.Param["claim"] = claims
}
