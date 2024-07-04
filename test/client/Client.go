package main

import (
	"context"
	"fmt"
	"github.com/zhengweiye/gowssocket"
	"sync"
)

var wsClient *gowssocket.WsClient

func main() {
	ctx, _ := context.WithCancel(context.Background())
	waitGroup := &sync.WaitGroup{}
	client, err := gowssocket.NewClient("test", "ws://127.0.0.1:8888/ws1", ctx, waitGroup, NewMyHandler())
	fmt.Println(client)
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

func NewMyHandler() gowssocket.ClientHandler {
	return MyHandler{}
}

func (m MyHandler) Connected(conn gowssocket.Connection) {
	fmt.Println("连接进来:", conn.Conn().RemoteAddr().String())
}

func (m MyHandler) Do(conn gowssocket.Connection, data gowssocket.HandlerData) error {
	fmt.Println(">>>Read: ", conn.ConnId(), ", 类型=", data.MessageType, ", 内容=", string(data.MessageData))
	return nil
}

func (m MyHandler) Disconnected(conn gowssocket.Connection) {
	fmt.Println(">>>Disconnected: ", conn.Conn().RemoteAddr().String(), ", isClose=", conn.IsClose())
}
