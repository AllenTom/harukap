package notification

import (
	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"sync"
)

type NotificationManager struct {
	Conns map[string]*NotificationConnection
	sync.Mutex
}

func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		Conns: make(map[string]*NotificationConnection),
	}
}

type NotificationConnection struct {
	Id         string
	Username   string
	Connection *websocket.Conn
	Logger     *logrus.Entry
	isClose    bool
}

func (m *NotificationManager) addConnection(conn *websocket.Conn, username string) *NotificationConnection {
	m.Lock()
	defer m.Unlock()
	id := xid.New().String()
	notification := &NotificationConnection{
		Connection: conn,
		Logger: WebsocketLogger.WithFields(logrus.Fields{
			"id":       id,
			"username": username,
		}),
		Username: username,
		Id:       id,
	}
	conn.SetCloseHandler(func(code int, text string) error {
		notification.isClose = true
		return nil
	})
	m.Conns[id] = notification
	return m.Conns[id]
}
func (m *NotificationManager) removeConnection(id string) {
	m.Lock()
	defer m.Unlock()
	delete(m.Conns, id)
}
func (m *NotificationManager) SendJSONToAll(data interface{}) {
	m.Lock()
	defer m.Unlock()
	for _, notificationConnection := range m.Conns {
		if notificationConnection.isClose {
			continue
		}
		err := notificationConnection.Connection.WriteJSON(data)
		if err != nil {
			notificationConnection.Logger.Error(err)
		}
	}
}
func (m *NotificationManager) SendJSONToUser(data interface{}, username string) {
	m.Lock()
	defer m.Unlock()
	for _, notificationConnection := range m.Conns {
		if notificationConnection.Username == username && !notificationConnection.isClose {
			err := notificationConnection.Connection.WriteJSON(data)
			if err != nil {
				notificationConnection.Logger.Error(err)
			}
		}
	}
}
