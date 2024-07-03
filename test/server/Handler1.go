package main

import (
	"fmt"
	"github.com/zhengweiye/gowssocket"
)

type Handler1 struct {
}

func NewHandler1() gowssocket.Handler {
	return Handler1{}
}

func (m Handler1) Connected(conn gowssocket.Connection) {
	fmt.Println("--Handler1: 第一次连接进来:", conn.Conn().RemoteAddr().String())
}

func (m Handler1) Read(conn gowssocket.Connection, data gowssocket.HandlerData) error {
	fmt.Println("--Handler1: 连接ID=", conn.Id(), ", 类型=", data.MessageType, ", 内容=", string(data.MessageData))
	conn.Send([]byte(fmt.Sprintf(">>>>%s", string(data.MessageData))))

	return nil
}

func (m Handler1) Disconnected(conn gowssocket.Connection) {
	if conn.IsClose() {
		fmt.Println("--Handler1: 连接断开:", conn.Conn().RemoteAddr().String(), ", isClose=", conn.IsClose())
	}
}

func (m Handler1) Error(conn gowssocket.Connection, err any) {
	fmt.Println("Handler1: 连接ID：", conn.Conn().RemoteAddr().String(), ", 异常：", err)
}
