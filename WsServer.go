package gowssocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type WsServer struct {
	Name                string
	upgrader            *websocket.Upgrader
	connectionManager   *ConnectionManager
	handler             Handler
	handlerOnce         sync.Once
	connQueueLength     int
	connQueueLengthOnce sync.Once
	heartbeatPeriod     *time.Duration
	heartbeatPeriodOnce sync.Once
	stopSignal          chan os.Signal
	workerPool          *Pool
}

var servers map[string]*WsServer
var serverLock sync.RWMutex

func init() {
	servers = make(map[string]*WsServer)
}

func NewWsServer(name string) *WsServer {
	serverLock.RLock()
	server, ok := servers[name]
	if ok {
		serverLock.RUnlock()
		return server
	}

	// 释放读锁，并且重新上写锁--->防止多个协程都获取不到server,往下执行
	serverLock.RUnlock()
	serverLock.Lock()
	defer serverLock.Unlock()

	fmt.Println(name, "---NewServer........")

	// 创建server
	workerPoolSize := runtime.NumCPU() << 2
	wsServer := &WsServer{
		Name: name,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connectionManager: newConnectionManager(),
		connQueueLength:   100,
		stopSignal:        make(chan os.Signal, 1),
		workerPool:        newPool(workerPoolSize),
	}

	// 信号量监听绑定
	wsServer.bindSignal()
	go wsServer.listenSignal()

	// 把server加入容器里面
	servers[name] = wsServer

	return wsServer
}

func (w *WsServer) SetConnQueueLength(length int) {
	w.connQueueLengthOnce.Do(func() {
		// 保证同一个url，每次请求,只赋值一次
		w.connQueueLength = length
	})
}

func (w *WsServer) SetHandler(handler Handler) {
	w.handlerOnce.Do(func() {
		fmt.Println(w.Name, "---SetHandler....")
		// 保证同一个url，每次请求,只赋值一次
		w.handler = handler
	})
}

func (w *WsServer) SetHeartbeatPeriod(period time.Duration) {
	w.heartbeatPeriodOnce.Do(func() {
		// 保证同一个url，每次请求,只赋值一次
		w.heartbeatPeriod = &period
	})
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

/**
 * （1）一个项目集成gowssocket，可能存在多个server（url）
 * （2）服务停止时，各个server都能监听到，然后各个服务都得等自己的业务执行完成
 */
func (w *WsServer) bindSignal() {
	signal.Notify(w.stopSignal,
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
}

func (w *WsServer) listenSignal() {
	for {
		select {
		case cmd := <-w.stopSignal:
			fmt.Println("signal:", cmd.String())
			switch cmd {
			case os.Interrupt:
				w.Shutdown()
			case os.Kill:
				w.Shutdown()
			case syscall.SIGTERM:
				w.Shutdown()
			}
		}
	}
}

func (w *WsServer) Shutdown() {
	fmt.Println(w.Name, ", 接受到关闭信号, 正在退出中.....")

	// 停止监听信号量
	signal.Stop(w.stopSignal)

	//TODO 堵塞等业务执行完成

	// 通知readLoop()和writeLoop()退出
	for _, conn := range w.connectionManager.GetAllConnections() {
		conn.Close(fmt.Errorf("server.Shutdown()被执行"))
	}

	// 回收ConnectionManager资源
	w.connectionManager.shutdown()

	// 回收线程池资源
	w.workerPool.shutdown()

	// 从servers移除server
	serverLock.Lock()
	defer serverLock.Unlock()
	delete(servers, w.Name)

	//TODO 停止web服务（net/http 或者 gin等web服务）

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
