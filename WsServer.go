package gowssocket

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/zhengweiye/gopool"
	"github.com/zhengweiye/goschedule"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type WsServer struct {
	name              string              // 服务名称
	upgrader          *websocket.Upgrader // 服务升级
	handler           Handler             // 特务处理器
	connectionManager *ConnectionManager  // 连接管理器
	pool              *gopool.Pool        // 协程池
	timer             *goschedule.Timer
	ctx               context.Context
	innerWaitGroup    *sync.WaitGroup
	globalWaitGroup   *sync.WaitGroup
	quitChan          chan bool
	isShutdown        bool
}

var servers map[string]*WsServer
var serverLock sync.RWMutex

func init() {
	servers = make(map[string]*WsServer)
}

/**
 * （1）一个业务系统可以创建多个Server，每种业务对应一个Server
 * （2）一个Server管理多个Connection
 * （3）通过group对Connection进行分组管理，例如：使用userId作为group，一个账号在不同pc建立Connection，通过userId作为group，可以批量给该分组进行推送数据
 */

func NewServer(name string, ctx context.Context, waitGroup *sync.WaitGroup, handler Handler) *WsServer {
	serverLock.RLock()
	server, ok := servers[name]
	if ok {
		serverLock.RUnlock()
		return server
	} else {
		serverLock.RUnlock()
	}

	// 释放读锁，并且重新上写锁--->防止多个协程都获取不到server,往下执行
	serverLock.Lock()
	defer serverLock.Unlock()

	// 创建server
	waitGroup.Add(1) //TODO 必须加1，在Shutdown时Done()
	pool := getPool(ctx, waitGroup)

	wsServer := &WsServer{
		name: name,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		handler:           handler,
		connectionManager: newConnectionManager(),
		pool:              pool,
		timer:             goschedule.NewTimer(pool, ctx, waitGroup),
		ctx:               ctx,
		innerWaitGroup:    &sync.WaitGroup{},
		globalWaitGroup:   waitGroup,
		quitChan:          make(chan bool),
	}

	// 把server加入容器里面
	servers[name] = wsServer

	// 监听服务
	go wsServer.listen()

	// 定时器
	wsServer.timer.Start()
	wsServer.timer.AddJob("heartBean", "心跳检测", false, time.Second, "@every 30s", wsServer.heartBeat, nil)

	fmt.Println(name, ">>> [WebSocket Server] 创建服务, 服务名称=", name)
	return wsServer
}

func (w *WsServer) Start(response http.ResponseWriter, request *http.Request, header http.Header) {
	// 分组ID
	url, err := url.Parse(request.RequestURI)
	if err != nil {
		panic(err)
	}
	group := url.Query().Get("group")
	if len(group) == 0 {
		fmt.Println(">>> [WebSocket Server] url缺少group参数")
		panic("url缺少group参数")
	}

	// 服务升级
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
	conn, err := w.upgrader.Upgrade(response, request, nil)
	if err != nil {
		panic(err)
	}

	// 创建connection
	wsConn := newConnectionServer(group, conn.RemoteAddr().String(), conn, w.handler, w, w.pool, w.innerWaitGroup)

	// 加入ConnectionManager
	w.connectionManager.add(wsConn)
}

func (w *WsServer) listen() {
	for {
		select {
		case <-w.quitChan:
			fmt.Printf(">>> [WebSocket Server] [%s] 退出监听...\n", w.name)
			return
		case <-w.ctx.Done():
			fmt.Printf(">>> [WebSocket Server] [%s] 接受到Context取消信号...\n", w.name)
			if !w.isShutdown {
				w.Shutdown()
			}
		}
	}
}

func (w *WsServer) Shutdown() {
	w.isShutdown = true
	fmt.Printf(">>> [WebSocket Server] [%s] 正在停止====================================\n", w.name)

	// 等待任务执行完成
	w.innerWaitGroup.Wait()

	// 释放资源
	w.release()

	// 退出监听
	close(w.quitChan)

	// 通知业务系统的http服务监听
	w.globalWaitGroup.Done()
	fmt.Printf(">>> [WebSocket Server] [%s] 完成停止====================================\n", w.name)
}

func (w *WsServer) release() {
	// 通知readLoop()和writeLoop()退出
	for _, conn := range w.connectionManager.GetAllConnections() {
		conn.Close()
	}

	// 从servers移除server
	serverLock.Lock()
	defer serverLock.Unlock()
	delete(servers, w.name)

	//TODO 回收线程池资源-->如果协程池不是共用，则可以关闭
	//TODO 疑问：server如何持续不断的监听客户端进来的呢？
	//TODO 客户端一个client对应多个conn?那么怎么保存conn
	//TODO 一些通用的东西

	//w.workerPool.Shutdown()
}

func (w *WsServer) BroadcastToAll(data []byte) {
	conns := w.connectionManager.GetAllConnections()
	if len(conns) > 0 {
		for _, conn := range conns {
			conn.Send(data)
		}
	}
}

func (w *WsServer) BroadcastToGroup(groupId string, data []byte) {
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

func (w *WsServer) GetServerName() string {
	return w.name
}

func (w *WsServer) heartBeat(param map[string]any) (err error, result string) {
	/**
	 * 逻辑如下：
	 * （1）conn的属性（lastContactTime），发送和接受数据时，更新该字段
	 * （2）定时器，遍历ConnectionManager，找出time.Now()-lastContactTime>60秒的conn
	 * （3）给conn发送信息，如果发现没有回应，则断开连接
	 */
	connectionList := w.connectionManager.GetAllConnections()
	for _, conn := range connectionList {
		defer func() {
			if err2 := recover(); err2 != nil {
				fmt.Printf(">>> [WebSocket Server] [%s] 心跳检查异常: %v\n", w.name, err)
			}
		}()

		// 超过90秒没有通信，才发送心跳包
		if time.Now().Sub(conn.LastTime()).Seconds() > 90 {
			err2 := conn.Conn().WriteMessage(websocket.PingMessage, []byte{})
			if err2 != nil {
				conn.Close()
			}
		}
	}
	return
}
