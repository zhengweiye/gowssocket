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

func (m Handler2) Connected(conn *gowssocket.Connection) {
	fmt.Println(conn.GetServerName(), "--Handler2: 第一次连接进来:", conn.Conn.RemoteAddr().String())
}

func (m Handler2) Read(conn *gowssocket.Connection, data gowssocket.WsData) error {
	fmt.Println(conn.GetServerName(), "--Handler2: 连接ID=", conn.ConnId, ", 类型=", data.Type, ", 内容=", string(data.Data))
	conn.Send(gowssocket.WsData{
		Type: data.Type,
		Data: []byte(fmt.Sprintf(">>>>%s", string(data.Data))),
	})

	return nil
}

func (m Handler2) Disconnected(conn *gowssocket.Connection, err error) {
	if conn.IsClose {
		fmt.Println(conn.GetServerName(), "--Handler2: 连接断开:", conn.Conn.RemoteAddr().String(), ", isClose=", conn.IsClose, ", 异常：", err)
	}
}

func (m Handler2) Error(conn *gowssocket.Connection, err any) {
	fmt.Println("Handler2: 连接ID：", conn.Conn.RemoteAddr().String(), ", 异常：", err)
}
