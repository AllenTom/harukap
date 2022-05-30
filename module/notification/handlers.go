package notification

import (
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
)

var WebsocketLogger = logrus.New().WithField("scope", "websocket")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
