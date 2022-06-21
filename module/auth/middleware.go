package auth

import (
	"github.com/allentom/haruka"
)

type AuthMiddleware struct {
	OnError       func(c *haruka.Context, err error)
	Module        *AuthModule
	RequestFilter func(c *haruka.Context) bool
}

func (m AuthMiddleware) OnRequest(c *haruka.Context) {
	if m.RequestFilter != nil && !m.RequestFilter(c) {
		return
	}
	jwtToken := m.Module.ParseAuthHeader(c)
	if m.Module.Config.EnableAnonymous && len(jwtToken) == 0 {
		return
	}
	claim, err := m.Module.ParseToken(jwtToken)
	if err != nil {
		m.OnError(c, err)
		return
	}
	c.Param["claim"] = claim
}
