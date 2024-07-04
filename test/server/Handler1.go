package main

import (
	"fmt"
	"github.com/zhengweiye/gowssocket"
	"net/url"
)

type Handler1 struct {
}

func NewHandler1() gowssocket.ServerHandler {
	return Handler1{}
}

func (m Handler1) Connected(conn gowssocket.Connection, connManager *gowssocket.ConnectionManager, url *url.URL) {
	fmt.Println("Handler1 Connected:", conn.Conn().RemoteAddr().String())
}

func (m Handler1) Do(conn gowssocket.Connection, data gowssocket.HandlerData) error {
	fmt.Println("Handler1 Read: 连接ID=", conn.ConnId(), ", 类型=", data.MessageType, ", 内容=", string(data.MessageData))
	conn.Send([]byte(fmt.Sprintf("handler1: %s", string(data.MessageData))))

	return nil
}

func (m Handler1) Disconnected(conn gowssocket.Connection, connManager *gowssocket.ConnectionManager) {
	if conn.IsClose() {
		fmt.Println("Handler1 Disconnected: 连接断开:", conn.Conn().RemoteAddr().String(), ", isClose=", conn.IsClose())
	}
}
