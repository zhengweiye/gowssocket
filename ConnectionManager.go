package gowssocket

import (
	"fmt"
	"strings"
	"sync"
)

type ConnectionManager struct {
	connections    map[string]Connection
	connectionLock sync.RWMutex
	split          string
}

/**
 * 每个服务对应一个ConnectionManager
 */
func newConnectionManager() *ConnectionManager {
	connectionManager := &ConnectionManager{
		connections: make(map[string]Connection),
		split:       "@zwy@",
	}
	return connectionManager
}

func (c *ConnectionManager) Add(conn Connection, connGroup, connId string) {
	if len(connGroup) == 0 {
		panic("连接分组为空")
	}
	if len(connId) == 0 {
		panic("连接Id为空")
	}
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	// 判断是否存在
	for key, _ := range c.connections {
		connName := strings.Split(key, c.split)[1]
		if connName == connId {
			panic("连接Id已经存在")
		}
	}

	// 加入容器
	key := fmt.Sprintf("%s%s%s", connGroup, c.split, connId)
	connectionServer := conn.(*ConnectionServer)
	connectionServer.setConnId(connId)
	connectionServer.setConnGroup(connGroup)
	c.connections[key] = connectionServer
	count := c.getCount(connGroup)
	fmt.Println(">>> [WebSocket Server] 添加连接, 服务名=", connectionServer.GetServerName(), ", 连接分组=", connGroup, ", 连接Id=", connId, ",  剩余连接数量=", count)
}

func (c *ConnectionManager) Remove(conn Connection) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	connectionServer := conn.(*ConnectionServer)
	key := fmt.Sprintf("%s%s%s", conn.ConnGroup(), c.split, conn.ConnId())
	delete(c.connections, key)
	count := c.getCount(conn.ConnGroup())

	fmt.Println(">>> [WebSocket Server] 移除连接, 服务名=", connectionServer.GetServerName(), ", 连接分组=", conn.ConnGroup(), ", 连接Id=", conn.ConnId(), ", 剩余连接数量=", count)
}

func (c *ConnectionManager) getCount(group string) int {
	count := 0
	for key, _ := range c.connections {
		groupName := strings.Split(key, c.split)[0]
		if groupName == group {
			count += 1
		}
	}
	return count
}

func (c *ConnectionManager) GetAllConnections() []Connection {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	conns := []Connection{}
	for _, value := range c.connections {
		conns = append(conns, value)
	}
	return conns
}

func (c *ConnectionManager) GetConnectionGroup(group string) []Connection {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	var list []Connection
	for key, value := range c.connections {
		groupName := strings.Split(key, c.split)[0]
		if groupName == group {
			list = append(list, value)
		}
	}
	return list
}

func (c *ConnectionManager) GetConnection(connId string) Connection {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	for key, value := range c.connections {
		connName := strings.Split(key, c.split)[1]
		if connName == connId {
			return value
		}
	}
	return nil
}
