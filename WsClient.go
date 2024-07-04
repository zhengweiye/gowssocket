package gowssocket

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/zhengweiye/gopool"
	"sync"
)

type WsClient struct {
	conn            Connection
	handler         ClientHandler
	pool            *gopool.Pool
	ctx             context.Context
	innerWaitGroup  *sync.WaitGroup
	globalWaitGroup *sync.WaitGroup
	quitChan        chan bool
	isShutdown      bool
}

/**
 * （1）一个业务系统可以创建多个Client，每种业务对应一个Client
 * （2）一个Client对应一个Connection
 */

func NewClient(group, url string, ctx context.Context, waitGroup *sync.WaitGroup, handler ClientHandler) (client *WsClient, err error) {
	pool := getPool(ctx, waitGroup)
	client = &WsClient{
		handler:         handler,
		pool:            pool,
		ctx:             ctx,
		innerWaitGroup:  &sync.WaitGroup{},
		globalWaitGroup: waitGroup,
		quitChan:        make(chan bool),
	}

	err = client.connect(url)
	return
}

func (w *WsClient) connect(url string) (err error) {
	// 连接服务端
	originConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return
	}

	// 创建connection
	wsConn := newConnectionClient(url, w, originConn, w.handler, w.pool, w.innerWaitGroup)
	w.conn = wsConn

	// 监听
	go w.listen()
	return
}

func (w *WsClient) listen() {
	for {
		select {
		case <-w.quitChan:
			fmt.Printf(">>> [WebSocket Client] 退出监听...\n")
			return
		case <-w.ctx.Done():
			fmt.Printf(">>> [WebSocket Client] 接受到Context取消信号...\n")
			if !w.isShutdown {
				w.Shutdown()
			}
		}
	}
}

func (w *WsClient) Shutdown() {
	w.isShutdown = true
	fmt.Printf(">>> [WebSocket Client] 正在停止====================================\n")

	// 等待任务执行完成
	w.innerWaitGroup.Wait()

	// 释放资源
	w.release()

	// 退出监听
	close(w.quitChan)

	// 通知业务系统的http服务监听
	w.globalWaitGroup.Done()
	fmt.Printf(">>> [WebSocket Client] 完成停止====================================\n")
}

func (w *WsClient) release() {
	// 通知readLoop()和writeLoop()退出
	w.conn.Close()

	//TODO 回收线程池资源-->如果协程池不是共用，则可以关闭
	//w.workerPool.Shutdown()
}

// err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
