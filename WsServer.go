package gowssocket

import (
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"time"
)

type WsServer struct {
	upgrader          *websocket.Upgrader
	connectionManager *ConnectionManager
	handler           Handler
	connQueueLength   int
	heartbeatPeriod   *time.Duration
}

/**
	ws := &websocket.Upgrader{
        ReadBufferSize:  4096,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            if r.Method != "GET" {
                fmt.Println("method is not GET")
                return false
            }
            if r.URL.Path != "/ws" {
                fmt.Println("path error")
                return false
            }
            return true
        },
    }
*/

func NewWsServer() *WsServer {
	return &WsServer{
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connectionManager: newConnectionManager(),
		connQueueLength:   100,
	}
}

func (w *WsServer) SetConnQueueLength(length int) {
	w.connQueueLength = length
}

func (w *WsServer) SetHandler(handler Handler) {
	w.handler = handler
}

func (w *WsServer) SetHeartbeatPeriod(period time.Duration) {
	w.heartbeatPeriod = &period
}

func (w *WsServer) Start(response http.ResponseWriter, request *http.Request, header http.Header) {
	// 分组ID
	url, err := url.Parse(request.RequestURI)
	if err != nil {
		panic(err)
	}
	groupId := url.Query().Get("groupId")
	if len(groupId) == 0 {
		panic("url缺少groupId参数")
	}

	// 服务升级
	conn, err := w.upgrader.Upgrade(response, request, nil)
	if err != nil {
		panic(err)
	}

	// 创建connection
	heartbeatPeriod := time.Second * 30
	if w.heartbeatPeriod != nil {
		heartbeatPeriod = *w.heartbeatPeriod
	}
	wsConn := newConnection(groupId, conn.RemoteAddr().String(), w.connQueueLength, &heartbeatPeriod, nil, conn, w.handler, w, nil)

	// 监听connection
	go wsConn.listen()

	// 加入ConnectionManager
	// TODO 方式一
	w.connectionManager.registerChan <- wsConn

	// TODO 方式二
	//w.connectionManager.add(groupId, wsConn)
}

func (w *WsServer) BroadcastToAll(data WsData) {
	conns := w.connectionManager.GetAllConnections()
	if len(conns) > 0 {
		for _, conn := range conns {
			conn.Send(data)
		}
	}
}

func (w *WsServer) BroadcastToGroup(groupId string, data WsData) {
	conns := w.connectionManager.GetConnectionGroup(groupId)
	if len(conns) > 0 {
		for _, conn := range conns {
			conn.Send(data)
		}
	}
}

func (w *WsServer) GetConnectionManager() *ConnectionManager {
	return w.connectionManager
}
