package main

import (
	"context"
	"fmt"
	"github.com/zhengweiye/gowssocket"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	waitGroup := &sync.WaitGroup{}
	handler1 := NewHandler1()
	handler2 := NewHandler2()

	http.HandleFunc("/ws1", func(writer http.ResponseWriter, request *http.Request) {
		wsServer := gowssocket.NewServer("ws1", ctx, waitGroup, handler1) // 每个url,无论多少次请求,只执行一次
		wsServer.Start(writer, request, request.Header)                   // 每个url,每次请求都会执行一次
	})

	http.HandleFunc("/ws2", func(writer http.ResponseWriter, request *http.Request) {
		wsServer := gowssocket.NewServer("ws2", ctx, waitGroup, handler2) // 每个url,无论多少次请求,只执行一次
		wsServer.Start(writer, request, request.Header)                   // 每个url,每次请求都会执行一次
	})

	httpServer := &http.Server{
		Addr: ":8888",
	}
	go func() {
		fmt.Println("准备启动服务....")
		err := httpServer.ListenAndServe()
		fmt.Println("启动服务异常：", err)
	}()
	// 等待中断信号，以优雅地关闭服务器
	serverChan = make(chan os.Signal)
	signal.Notify(serverChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGILL,
		syscall.SIGTRAP,
		syscall.SIGABRT,
		syscall.SIGBUS,
		syscall.SIGFPE,
		syscall.SIGKILL,
		syscall.SIGSEGV,
		syscall.SIGPIPE,
		syscall.SIGALRM,
		syscall.SIGTERM, //terminated ==> docker重启项目会触发该信号
	)
	signalCmd := <-serverChan
	switch signalCmd {
	case os.Interrupt:
		stop(ctx, cancel, waitGroup, httpServer)
	case os.Kill:
		stop(ctx, cancel, waitGroup, httpServer)
	case syscall.SIGTERM:
		stop(ctx, cancel, waitGroup, httpServer)
	}
}

var serverChan chan os.Signal

func stop(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, httpServer *http.Server) {
	// 停止监听信号量
	signal.Stop(serverChan)

	// 取消context
	cancel()

	// 等业务执行完成
	wg.Wait()

	// 关闭服务器：阻止新的连接进入并等待活跃连接处理完成后再终止程序，达到优雅退出的目的
	err := httpServer.Shutdown(context.Background())
	if err != nil {
		panic(err)
	}
}
