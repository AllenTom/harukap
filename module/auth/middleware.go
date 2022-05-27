package auth

import (
	"github.com/allentom/haruka"
)

type AuthMiddleware struct {
	OnError func(c *haruka.Context, err error)
	Module  *AuthModule
}

func (m AuthMiddleware) OnRequest(c *haruka.Context) {
	for _, path := range m.Module.NoAuthPath {
		if c.Request.URL.Path == path {
			return
		}
	}
	jwtToken := m.Module.ParseAuthHeader(c)
	if m.Module.Config.EnableAnonymous && len(jwtToken) == 0 {
		return
	}
	claim, err := m.Module.ParseToken(jwtToken)
	if err != nil {
		c.Interrupt()
		m.OnError(c, err)
		return
	}
	c.Param["claim"] = claim
}
