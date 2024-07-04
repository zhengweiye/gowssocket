package gowssocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/zhengweiye/gopool"
	"net/url"
	"sync"
	"time"
)

type ConnectionServer struct {
	server          *WsServer
	clientIp        string
	connGroup       string
	connId          string
	conn            *websocket.Conn //连接
	isClose         bool
	isCloseLock     sync.RWMutex
	quitChan        chan bool
	lastContactTime time.Time     //最新通信时间
	dataChan        chan wsData   //消息
	handler         ServerHandler //业务处理
	props           map[string]any
	propLock        sync.RWMutex
	pool            *gopool.Pool
	waitGroup       *sync.WaitGroup
}

type wsData struct {
	messageData []byte

	/**
	 * TextMessage = 1
	 * BinaryMessage = 2
	 * CloseMessage = 8
	 * PingMessage = 9
	 * PongMessage = 10
	 */
	messageType int
	waitGroup   *sync.WaitGroup
}

func newConnectionServer(
	url *url.URL,
	conn *websocket.Conn,
	handler ServerHandler,
	server *WsServer,
	pool *gopool.Pool,
	waitGroup *sync.WaitGroup,
) {
	fmt.Printf(">>> [WebSocket Server] 连接Id[%s]\n", conn.RemoteAddr().String())
	connWrap := &ConnectionServer{
		server:          server,
		clientIp:        conn.RemoteAddr().String(),
		conn:            conn,
		quitChan:        make(chan bool),
		lastContactTime: time.Now(),
		dataChan:        make(chan wsData, 1000),
		handler:         handler,
		props:           make(map[string]any),
		pool:            pool,
		waitGroup:       waitGroup,
	}

	// 回调函数执行
	connWrap.handler.Connected(connWrap, connWrap.server.connectionManager, url)

	// 连接监听
	connWrap.listen()
}

func (c *ConnectionServer) listen() {
	// 监听请求
	go c.readLoop()

	// 监听响应
	go c.writeLoop()
}

func (c *ConnectionServer) LastTime() time.Time {
	return c.lastContactTime
}

func (c *ConnectionServer) setConnId(id string) {
	c.connId = id
}

func (c *ConnectionServer) setConnGroup(group string) {
	c.connGroup = group
}

func (c *ConnectionServer) Conn() *websocket.Conn {
	return c.conn
}

func (c *ConnectionServer) ConnGroup() string {
	return c.connGroup
}

func (c *ConnectionServer) ConnId() string {
	return c.connId
}

func (c *ConnectionServer) Close() {
	c.isCloseLock.Lock()
	defer c.isCloseLock.Unlock()

	// 防止read()和write()重复往下执行
	if c.isClose {
		return
	}
	c.isClose = true

	// 通知read()和write()都退出
	//TODO read()异常-->调用Close()-->close(c.closeChan)-->write()监听到退出-->调用Close()....
	//TODO write()异常-->调用Close()-->close(c.closeChan)-->怎么通知read()也退出呢？
	close(c.quitChan) //TODO 必须先关闭这个

	// 关闭dataChan
	close(c.dataChan)

	// 关闭底层conn-->放到最后，因为正在处理的任务还可能需要响应给客户端
	defer func() {
		if err2 := recover(); err2 != nil {
			fmt.Printf(">>> [WebSocket Server] 客户端IP[%s], 断开连接触发回调异常: %v\n", c.clientIp, err2)
		}
		err := c.conn.Close()
		fmt.Printf(">>> [WebSocket Server] 客户端IP[%s], 关闭底层连接异常: %v\n", c.clientIp, err)
	}()
	// 回调函数执行
	c.handler.Disconnected(c, c.server.connectionManager)
}

func (c *ConnectionServer) IsClose() bool {
	return c.isClose
}

func (c *ConnectionServer) Send(data []byte) {
	c.isCloseLock.RLock()
	defer c.isCloseLock.RUnlock()
	if !c.isClose {
		//issue：如果连接断开，那么还调用该方法，则抛异常，导致服务停止
		//疑问：为啥这里使用defer捕捉异常，还是对外panic异常呢？
		c.dataChan <- wsData{
			messageData: data,
			messageType: websocket.TextMessage,
			waitGroup:   c.waitGroup,
		}
	}
}

func (c *ConnectionServer) writeLoop() {
	defer c.Close()
	for {
		select {
		case <-c.quitChan:
			return
		case data := <-c.dataChan:
			err := c.processResponse(data)
			if err != nil {
				fmt.Printf(">>> [WebSocket Server] 客户端IP[%s], 响应数据异常: %v\n", c.clientIp, err)
				return
			}
		}
	}
}

func (c *ConnectionServer) readLoop() {
	defer c.Close()
	for {
		select {
		case <-c.quitChan:
			return
		default:
			//TODO 堵塞，等待对方发送数据，通过查看源码得知，不能接受PingMessage和PongMessage
			messageType, data, err := c.conn.ReadMessage()
			if err != nil {
				fmt.Printf(">>> [WebSocket Server] 客户端IP[%s], 读取数据异常: %v\n", c.clientIp, err)
				return
			}

			// 更新最新通信时间
			c.lastContactTime = time.Now()

			// 处理数据
			c.pool.ExecTask(gopool.Job{
				JobName: "websocketProcess",
				JobFunc: c.processRequest,
				JobParam: map[string]any{
					"data": wsData{
						messageType: messageType,
						messageData: data,
						waitGroup:   c.waitGroup,
					},
				},
			})
		}
	}
}

func (c *ConnectionServer) processResponse(data wsData) (err error) {
	data.waitGroup.Add(1)
	defer data.waitGroup.Done()

	// 往客户端推送数据--->不能使用子协程去执行，否则会出现粘包现象
	err = c.conn.WriteMessage(data.messageType, data.messageData)
	if err != nil {
		return
	}

	// 更新最新通信时间
	c.lastContactTime = time.Now()
	return
}

func (c *ConnectionServer) processRequest(workerId int, jobName string, param map[string]any) (err error) {
	data := param["data"].(wsData)
	data.waitGroup.Add(1)
	defer data.waitGroup.Done()

	defer func() {
		if err2 := recover(); err2 != nil {
			fmt.Printf(">>> [WebSocket Server] 客户端IP[%s], 处理业务时异常：%v\n", c.clientIp, err)
		}
	}()

	err = c.handler.Do(c, HandlerData{
		MessageData: data.messageData,
		MessageType: data.messageType,
	})
	if err != nil {
		panic(err)
	}
	return
}

func (c *ConnectionServer) Reconnect() {

}

func (c *ConnectionServer) SetProp(key string, value any) error {
	c.propLock.Lock()
	defer c.propLock.Unlock()

	_, ok := c.props[key]
	if ok {
		return fmt.Errorf("属性key=%s已经存在", key)
	}
	c.props[key] = value
	return nil
}

func (c *ConnectionServer) GetProp(key string) any {
	c.propLock.RLock()
	defer c.propLock.RUnlock()

	value, ok := c.props[key]
	if !ok {
		return nil
	}
	return value
}

func (c *ConnectionServer) DelProp(key string) {
	c.propLock.Lock()
	defer c.propLock.Unlock()
	delete(c.props, key)
}

func (c *ConnectionServer) GetConnectionManager() *ConnectionManager {
	return c.server.connectionManager
}

func (c *ConnectionServer) GetServerName() string {
	return c.server.name
}
