package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/zhengweiye/gowssocket"
	"strconv"
	"time"
)

var wsClient *gowssocket.WsClient

func main() {
	wsClient = gowssocket.NewWsClient()
	wsClient.SetHandler(NewMyHandler())
	err := wsClient.Connect("ws://127.0.0.1:8888/ws?groupId=123")
	if err != nil {
		panic(err)
	}
	/*fmt.Println("========================1>", wsClient.GetConnection().IsClose)
	time.Sleep(5 * time.Second)

	wsClient.GetConnection().Send(gowssocket.WsData{
		Type: 1,
		Data: []byte("hello world1"),
	})

	wsClient.GetConnection().Close(nil)

	fmt.Println("========================2>", wsClient.GetConnection().IsClose)
	time.Sleep(5 * time.Second)
	fmt.Println("========================3>", wsClient.GetConnection().IsClose)

	wsClient.GetConnection().Send(gowssocket.WsData{
		Type: 1,
		Data: []byte("hello world2"),
	})*/

	for {
	}
}

type MyHandler struct {
}

func NewMyHandler() gowssocket.Handler {
	return MyHandler{}
}

func (m MyHandler) Connected(conn *gowssocket.Connection) {
	fmt.Println("连接进来:", conn.Conn.RemoteAddr().String())

	for i := 0; i < 5; i++ {
		conn.Send(gowssocket.WsData{
			Type: websocket.TextMessage,
			Data: []byte(strconv.Itoa(i)),
		})

		time.Sleep(8 * time.Second)
	}
}

func (m MyHandler) Read(conn *gowssocket.Connection, data gowssocket.WsData) error {
	fmt.Println(">>>Read: ", conn.ConnId, ", 类型=", data.Type, ", 内容=", string(data.Data))
	return nil
}

func (m MyHandler) Disconnected(conn *gowssocket.Connection, err error) {
	fmt.Println(">>>Disconnected: ", conn.Conn.RemoteAddr().String(), ", isClose=", conn.IsClose, ", 异常：", err)

	conn.Reconnect()
}

func (m MyHandler) Error(conn *gowssocket.Connection, err any) {
	fmt.Println(">>>Error: ", conn.Conn.RemoteAddr().String(), ", 异常：", err)
}
