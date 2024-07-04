package gowssocket

import "net/url"

type ServerHandler interface {
	Connected(conn Connection, connManager *ConnectionManager, url *url.URL)
	Do(conn Connection, data HandlerData) error
	Disconnected(conn Connection, connManager *ConnectionManager)
}

type ClientHandler interface {
	Connected(conn Connection)
	Do(conn Connection, data HandlerData) error
	Disconnected(conn Connection)
}

type HandlerData struct {
	MessageType int    `json:"messageType"`
	MessageData []byte `json:"messageData"`
}
