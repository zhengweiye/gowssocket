package main

import (
	"fmt"
	"github.com/zhengweiye/gowssocket"
	"net/http"
	"sync"
)

var wsServer *gowssocket.WsServer
var wsOnce sync.Once

func main() {
	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		wsOnce.Do(func() {
			wsServer = gowssocket.NewWsServer()
			wsServer.SetHandler(NewMyHandler())
		})
		wsServer.Start(writer, request, request.Header)
	})

	http.ListenAndServe(":8888", nil)
}

type MyHandler struct {
}

func NewMyHandler() gowssocket.Handler {
	return MyHandler{}
}

func (m MyHandler) Connected(conn *gowssocket.Connection) {
	fmt.Println("第一次连接进来:", conn.Conn.RemoteAddr().String())
}

func (m MyHandler) Read(conn *gowssocket.Connection, data gowssocket.WsData) error {
	fmt.Println("连接ID=", conn.ConnId, ", 类型=", data.Type, ", 内容=", string(data.Data))
	conn.Send(gowssocket.WsData{
		Type: data.Type,
		Data: []byte(fmt.Sprintf(">>>>%s", string(data.Data))),
	})

	return nil
}

func (m MyHandler) Disconnected(conn *gowssocket.Connection, err error) {
	if conn.IsClose {
		fmt.Println("连接断开:", conn.Conn.RemoteAddr().String(), ", isClose=", conn.IsClose, ", 异常：", err)
	}
}

func (m MyHandler) Error(conn *gowssocket.Connection, err any) {
	fmt.Println(">>>>>连接ID：", conn.Conn.RemoteAddr().String(), ", 异常：", err)
}
