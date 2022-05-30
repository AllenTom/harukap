package notification

import (
	"github.com/allentom/haruka"
	"github.com/gorilla/websocket"
)

type NotificationModule struct {
	NotificationSocketHandler haruka.RequestHandler
	Manager                   *NotificationManager
}

func (m *NotificationModule) InitModule() error {
	m.Manager = NewNotificationManager()
	m.NotificationSocketHandler = func(context *haruka.Context) {
		c, err := upgrader.Upgrade(context.Writer, context.Request, nil)
		if err != nil {
			WebsocketLogger.Error(err)
			return
		}
		notifier := m.Manager.addConnection(c, context.Param["username"].(string))
		defer func() {
			m.Manager.removeConnection(notifier.Id)
			c.Close()
		}()
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, 1005, 1000) {
					notifier.Logger.Error(err)
				}
				break
			}
		}
	}
	return nil
}
