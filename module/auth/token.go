package auth

import (
	"errors"
	"github.com/allentom/haruka"
	"github.com/allentom/harukap/commons"
	"github.com/dgrijalva/jwt-go"
	"strings"
)

func (m *AuthModule) ParseAuthHeader(c *haruka.Context) string {
	jwtToken := ""
	jwtToken = c.Request.Header.Get("Authorization")
	if len(jwtToken) == 0 {
		jwtToken = c.GetQueryString("a")
	}
	if len(jwtToken) == 0 {
		jwtToken = c.GetQueryString("token")
	}
	jwtToken = strings.TrimPrefix(jwtToken, "Bearer ")
	return jwtToken
}

func (m *AuthModule) ParseToken(jwtToken string) (commons.AuthUser, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(jwtToken, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	mapClaims := token.Claims.(jwt.MapClaims)
	isu := mapClaims["iss"].(string)
	authPlugin := m.GetAuthPluginByName(isu)
	if authPlugin == nil {
		return nil, errors.New("auth plugin not found")
	}
	authUser, err := authPlugin.GetAuthUserByToken(jwtToken)
	if err != nil {
		return nil, err
	}
	return authUser, nil
}
