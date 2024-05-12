package gowssocket

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/zhengweiye/gopool"
	"sync"
	"time"
)

type Connection struct {
	wsServer        *WsServer
	wsClient        *WsClient
	GroupId         string
	ConnId          string
	Conn            *websocket.Conn //连接
	IsClose         bool
	isCloseLock     sync.RWMutex
	lastContactTime time.Time      //最新通信时间
	heartbeatPeriod *time.Duration //心跳时间
	reconnectPeriod *time.Duration
	dataChan        chan WsData //消息
	handler         Handler     //业务处理
	closeChan       chan byte
	props           map[string]any
	propLock        sync.RWMutex
}

type WsData struct {
	Data []byte
	Type int
}

func newConnection(groupId, connId string, connQueueLength int, heartbeatPeriod, reconnectPeriod *time.Duration, conn *websocket.Conn, handler Handler, wsServer *WsServer, wsClient *WsClient) *Connection {
	return &Connection{
		GroupId:         groupId,
		ConnId:          connId,
		Conn:            conn,
		IsClose:         false,
		lastContactTime: time.Now(),
		heartbeatPeriod: heartbeatPeriod,
		reconnectPeriod: reconnectPeriod,
		dataChan:        make(chan WsData, connQueueLength),
		closeChan:       make(chan byte),
		handler:         handler,
		wsServer:        wsServer,
		wsClient:        wsClient,
		props:           make(map[string]any),
	}
}

func (c *Connection) listen() {
	// 连接成功，触发函数
	if c.handler != nil {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					fmt.Println("连接成功触发函数异常:", err)
				}
			}()
			c.handler.Connected(c)
		}()
	}

	// 监听请求
	go c.readLoop()

	// 监听响应
	go c.writeLoop()

	// 心跳检测
	if c.wsServer != nil {
		timer(1*time.Second, *c.heartbeatPeriod, c.heartBeatFunc)
	}
}

func (c *Connection) Close(err error) {
	c.isCloseLock.Lock()
	defer c.isCloseLock.Unlock()

	// 防止read()和write()重复往下执行
	if c.IsClose {
		return
	}

	// 关闭底层conn
	c.Conn.Close()
	c.IsClose = true

	// 从连接管理器移除
	if c.wsServer != nil {
		//TODO 方式一
		c.wsServer.connectionManager.unregisterChan <- c

		//TODO 方式二
		//c.wsServer.connectionManager.remove(c.ConnId)
	}

	// 通知read()和write()都退出
	//TODO read()异常-->调用Close()-->close(c.closeChan)-->write()监听到退出-->调用Close()....
	//TODO write()异常-->调用Close()-->close(c.closeChan)-->怎么通知read()也退出呢？
	close(c.closeChan) //TODO 必须先关闭这个

	// 关闭dataChan
	close(c.dataChan)

	// 触发断开函数
	if c.handler != nil {
		go func() {
			defer func() {
				if err2 := recover(); err2 != nil {
					fmt.Println("触发断开连接函数异常:", err2)
				}
			}()
			c.handler.Disconnected(c, err)
		}()
	}
}

func (c *Connection) Send(data WsData) {
	c.isCloseLock.RLock()
	defer c.isCloseLock.RUnlock()
	if !c.IsClose {
		//issue：如果连接断开，那么还调用该方法，则抛异常，导致服务停止
		//疑问：为啥这里使用defer捕捉异常，还是对外panic异常呢？
		c.dataChan <- data
	}
}

func (c *Connection) writeLoop() {
	defer c.Close(errors.New("writeLoop()异常,退出chan监听"))
	for {
		select {
		case data := <-c.dataChan:
			//TODO 注意：这里不能使用子协程去执行，否则会出现粘包现象
			//TODO 但是业务端调用send()时可以子协程，因为这里使用chan，是顺序执行的
			err := c.Conn.WriteMessage(data.Type, data.Data)
			if err != nil {
				fmt.Println("connection#writeLoop()异常:", err)
				return
			}

			// 更新最新通信时间
			if c.wsServer != nil {
				c.lastContactTime = time.Now()
			}
		case <-c.closeChan:
			return
		}
	}
}

func (c *Connection) readLoop() {
	defer c.Close(errors.New("readLoop()异常,退出chan监听"))

	for {
		select {
		case <-c.closeChan:
			return
		default:
			// 通过查看源码得知，不能接受PingMessage和PongMessage
			messageType, data, err := c.Conn.ReadMessage()
			if err != nil {
				fmt.Println("connection#readLoop()异常: ", err)
				return
			}

			// 更新最新通信时间
			if c.wsServer != nil {
				c.lastContactTime = time.Now()
			}

			// 处理数据
			if c.handler != nil {
				var workerPool *gopool.Pool
				if c.wsServer != nil {
					workerPool = c.wsServer.workerPool
				} else if c.wsClient != nil {
					workerPool = c.wsClient.workerPool
				}
				if workerPool != nil {
					jobFunc := func(workId int, param map[string]any) (err error) {
						defer func() {
							if err2 := recover(); err2 != nil {
								c.handler.Error(c, err2)
							}
						}()
						err = c.handler.Read(c, WsData{
							Type: messageType,
							Data: data,
						})
						if err != nil {
							c.handler.Error(c, err)
						}
						return
					}
					workerPool.ExecTask(gopool.Job{
						JobName:   "connection处理",
						JobFunc:   jobFunc,
						WaitGroup: nil,
						JobParam:  nil,
					})
				}
			}
		}
	}
}

func (c *Connection) Reconnect() {
	c.isCloseLock.Lock()
	defer c.isCloseLock.Unlock()
	if c.IsClose && c.wsClient != nil {
		ticker := time.NewTicker(*c.reconnectPeriod)
		for {
			select {
			case <-ticker.C:
				err := c.wsClient.Connect(c.wsClient.url) // 其实还是共用同一个wsClient,只是里面的conn更改了而已
				if err == nil {
					ticker.Stop()
					return
				}
			}
		}
	}
}

func (c *Connection) SetProp(key string, value any) error {
	c.propLock.Lock()
	defer c.propLock.Unlock()

	_, ok := c.props[key]
	if ok {
		return fmt.Errorf("属性key=%s已经存在", key)
	}
	c.props[key] = value
	return nil
}

func (c *Connection) GetProp(key string) any {
	c.propLock.RLock()
	defer c.propLock.RUnlock()

	value, ok := c.props[key]
	if !ok {
		return nil
	}
	return value
}

func (c *Connection) DelProp(key string) {
	c.propLock.Lock()
	defer c.propLock.Unlock()
	delete(c.props, key)
}

func (c *Connection) GetConnectionManager() *ConnectionManager {
	if c.wsClient != nil {
		return nil
	}
	return c.wsServer.connectionManager
}

func (c *Connection) GetServerName() string {
	if c.wsClient != nil {
		return ""
	}
	return c.wsServer.Name
}

func (c *Connection) heartBeatFunc() {
	/**
	 * 逻辑如下：
	 * （1）conn的属性（lastContactTime），发送和接受数据时，更新该字段
	 * （2）定时器，遍历ConnectionManager，找出time.Now()-lastContactTime>60秒的conn
	 * （3）给conn发送信息，如果发现没有回应，则断开连接
	 */
	connectionList := c.wsServer.connectionManager.GetAllConnections()
	for _, conn := range connectionList {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("connection#heartBeatFunc()异常:", err)
			}
		}()
		if time.Now().Sub(conn.lastContactTime).Seconds() > 30 {
			err := conn.Conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				c.Close(fmt.Errorf("检测不到[connId=%s]的心跳,服务端主动断开", conn.ConnId))
			}
		}
	}
}
