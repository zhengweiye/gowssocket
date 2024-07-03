package main

import (
	"fmt"
	"github.com/zhengweiye/gowssocket"
)

type Handler2 struct {
}

func NewHandler2() gowssocket.Handler {
	return Handler2{}
}

func (m Handler2) Connected(conn gowssocket.Connection) {
	fmt.Println("--Handler2: 第一次连接进来:", conn.Conn().RemoteAddr().String())
}

func (m Handler2) Read(conn gowssocket.Connection, data gowssocket.HandlerData) error {
	fmt.Println("--Handler2: 连接ID=", conn.Id(), ", 类型=", data.MessageType, ", 内容=", string(data.MessageData))
	conn.Send([]byte(fmt.Sprintf(">>>>%s", string(data.MessageData))))

	return nil
}

func (m Handler2) Disconnected(conn gowssocket.Connection) {
	if conn.IsClose() {
		fmt.Println("--Handler2: 连接断开:", conn.Conn().RemoteAddr().String(), ", isClose=", conn.IsClose())
	}
}

func (m Handler2) Error(conn gowssocket.Connection, err any) {
	fmt.Println("Handler2: 连接ID：", conn.Conn().RemoteAddr().String(), ", 异常：", err)
}
