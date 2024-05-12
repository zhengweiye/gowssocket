package gowssocket

import (
	"github.com/gorilla/websocket"
	"github.com/zhengweiye/gopool"
	"time"
)

type WsClient struct {
	conn            *Connection
	handler         Handler
	connQueueLength int
	url             string
	reconnectPeriod *time.Duration
	workerPool      *gopool.Pool
}

func NewWsClient() *WsClient {
	//workerPoolSize := runtime.NumCPU() << 2
	return &WsClient{
		workerPool: getPool(),
	}
}

/*
*
interrupt := make(chan os.Signal, 1)

	signal.Notify(interrupt, os.Interrupt)

case <-interrupt:

	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
	    return
	}
	select {
	    case <-done:
	    case <-time.After(time.Second):
	}
	return
*/

func (w *WsClient) Connect(url string) error {
	if len(url) == 0 {
		panic("连接地址为空")
	}
	w.url = url

	// 连接服务端
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}

	// 创建connection
	reconnectPeriod := time.Second * 2
	if w.reconnectPeriod != nil {
		reconnectPeriod = *w.reconnectPeriod
	}
	wsConn := newConnection("", conn.RemoteAddr().String(), w.connQueueLength, nil, &reconnectPeriod, conn, w.handler, nil, w)
	w.conn = wsConn

	// 监听connection
	go wsConn.listen()

	return nil
}

func (w *WsClient) SetConnQueueLength(length int) {
	w.connQueueLength = length
}

func (w *WsClient) SetHandler(handler Handler) {
	w.handler = handler
}

func (w *WsClient) SetReconnectPeriod(period time.Duration) {
	w.reconnectPeriod = &period
}

func (w *WsClient) GetConnection() *Connection {
	return w.conn
}
