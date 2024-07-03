package gowssocket

import (
	"github.com/gorilla/websocket"
	"time"
)

type Connection interface {
	Id() string
	Group() string
	LastTime() time.Time
	Conn() *websocket.Conn
	Close()
	IsClose() bool
	Send(data []byte)
	Reconnect()
	SetProp(key string, value any) error
	GetProp(key string) any
	DelProp(key string)
}
