package main

import (
	"github.com/zhengweiye/gowssocket"
	"net/http"
)

func main() {
	http.HandleFunc("/ws1", func(writer http.ResponseWriter, request *http.Request) {
		wsServer := gowssocket.NewWsServer("ws1")       //TODO 每个url,无论多少次请求,只执行一次
		wsServer.SetHandler(NewHandler1())              //TODO 每个url,无论多少次请求,只执行一次
		wsServer.Start(writer, request, request.Header) //TODO 每个url,每次请求都会执行一次
	})

	http.HandleFunc("/ws2", func(writer http.ResponseWriter, request *http.Request) {
		wsServer := gowssocket.NewWsServer("ws2")       //TODO 每个url,无论多少次请求,只执行一次
		wsServer.SetHandler(NewHandler2())              //TODO 每个url,无论多少次请求,只执行一次
		wsServer.Start(writer, request, request.Header) //TODO 每个url,每次请求都会执行一次
	})

	http.ListenAndServe(":8888", nil)
}
