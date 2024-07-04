package gowssocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/zhengweiye/gopool"
	"sync"
	"time"
)

type ConnectionClient struct {
	client          *WsClient
	url             string
	conn            *websocket.Conn //连接
	isClose         bool
	isCloseLock     sync.RWMutex
	quitChan        chan bool
	lastContactTime time.Time     //最新通信时间
	dataChan        chan wsData   //消息
	handler         ClientHandler //业务处理
	props           map[string]any
	propLock        sync.RWMutex
	pool            *gopool.Pool
	waitGroup       *sync.WaitGroup
}

func newConnectionClient(url string,
	client *WsClient,
	conn *websocket.Conn,
	handler ClientHandler,
	pool *gopool.Pool,
	waitGroup *sync.WaitGroup,
) Connection {
	return &ConnectionClient{
		client:          client,
		url:             url,
		conn:            conn,
		quitChan:        make(chan bool),
		lastContactTime: time.Now(),
		dataChan:        make(chan wsData, 1000),
		handler:         handler,
		props:           make(map[string]any),
		pool:            pool,
		waitGroup:       waitGroup,
	}
}

func (c *ConnectionClient) LastTime() time.Time {
	return c.lastContactTime
}

func (c *ConnectionClient) Conn() *websocket.Conn {
	return c.conn
}

func (c *ConnectionClient) ConnGroup() string {
	return ""
}

func (c *ConnectionClient) ConnId() string {
	return ""
}

func (c *ConnectionClient) Close() {
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

	// 回调函数执行
	go func() {
		defer func() {
			if err2 := recover(); err2 != nil {
				fmt.Printf(">>> [WebSocket Client] 断开连接触发回调异常: %v\n", err2)
			}
		}()
		c.handler.Disconnected(c)
	}()

	// 关闭底层conn-->放到最后，因为正在处理的任务还可能需要响应给客户端
	c.conn.Close()
}

func (c *ConnectionClient) IsClose() bool {
	return c.isClose
}

func (c *ConnectionClient) Send(data []byte) {
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

//TODO 有bug

func (c *ConnectionClient) Reconnect() {
	c.isCloseLock.Lock()
	defer c.isCloseLock.Unlock()

	if c.isClose {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				err := c.client.connect(c.url) // 其实还是共用同一个wsClient,只是里面的conn更改了而已
				if err == nil {
					ticker.Stop()
					return
				}
			}
		}
	}
}

func (c *ConnectionClient) SetProp(key string, value any) error {
	c.propLock.Lock()
	defer c.propLock.Unlock()

	_, ok := c.props[key]
	if ok {
		return fmt.Errorf("属性key=%s已经存在", key)
	}
	c.props[key] = value
	return nil
}

func (c *ConnectionClient) GetProp(key string) any {
	c.propLock.RLock()
	defer c.propLock.RUnlock()

	value, ok := c.props[key]
	if !ok {
		return nil
	}
	return value
}

func (c *ConnectionClient) DelProp(key string) {
	c.propLock.Lock()
	defer c.propLock.Unlock()
	delete(c.props, key)
}
