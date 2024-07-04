package gowssocket

import (
	"github.com/gorilla/websocket"
	"time"
)

type Connection interface {
	LastTime() time.Time
	Conn() *websocket.Conn
	ConnGroup() string
	ConnId() string
	Close()
	IsClose() bool
	Send(data []byte)
	Reconnect()
	SetProp(key string, value any) error
	GetProp(key string) any
	DelProp(key string)
}
