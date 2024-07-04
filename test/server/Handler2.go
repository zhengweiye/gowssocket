package main

import (
	"fmt"
	"github.com/zhengweiye/gowssocket"
	"net/url"
)

type Handler2 struct {
}

func NewHandler2() gowssocket.ServerHandler {
	return Handler2{}
}

func (m Handler2) Connected(conn gowssocket.Connection, connManager *gowssocket.ConnectionManager, url *url.URL) {
	fmt.Println("Handler2 Connected:", conn.Conn().RemoteAddr().String())
}

func (m Handler2) Do(conn gowssocket.Connection, data gowssocket.HandlerData) error {
	fmt.Println("Handler2 Read: 连接ID=", conn.ConnId(), ", 类型=", data.MessageType, ", 内容=", string(data.MessageData))
	conn.Send([]byte(fmt.Sprintf("handler2: %s", string(data.MessageData))))

	return nil
}

func (m Handler2) Disconnected(conn gowssocket.Connection, connManager *gowssocket.ConnectionManager) {
	if conn.IsClose() {
		fmt.Println("Handler2 Disconnected:", conn.Conn().RemoteAddr().String(), ", isClose=", conn.IsClose())
	}
}
