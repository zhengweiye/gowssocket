package gowssocket

import (
	"fmt"
	"sync"
)

type ConnectionManager struct {
	connections    map[string][]Connection
	connectionLock sync.RWMutex
}

/**
 * 每个服务对应一个ConnectionManager
 */
func newConnectionManager() *ConnectionManager {
	connectionManager := &ConnectionManager{
		connections: make(map[string][]Connection),
	}
	return connectionManager
}

func (c *ConnectionManager) add(conn Connection) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	connectionList, ok := c.connections[conn.Group()]
	if !ok {
		connectionList = []Connection{}
	}

	exist := false
	for _, connection := range connectionList {
		if connection.Id() == conn.Id() {
			exist = true
			break
		}
	}
	if !exist {
		c.connections[conn.Group()] = append(connectionList, conn)
	}
	connectionServer := conn.(*ConnectionServer)
	fmt.Println(">>> [WebSocket Server] 添加连接, 服务名=", connectionServer.GetServerName(), ", groupId=", conn.Group(), ", connId=", conn.Id(), ", exist=", exist, ", 连接数量=", len(c.connections[conn.Group()]))
}

func (c *ConnectionManager) remove(conn Connection) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()
	connectionServer := conn.(*ConnectionServer)
	for groupId, value := range c.connections {
		delIndex := -1
		for index, item := range value {
			if item.Id() == conn.Id() {
				delIndex = index
				break
			}
		}
		if delIndex > -1 {
			connList := append(value[:delIndex], value[delIndex+1:]...)
			c.connections[groupId] = connList
			fmt.Println(">>> [WebSocket Server] 移除连接, 服务名=", connectionServer.GetServerName(), ", groupId=", groupId, ", connId=", conn.Id(), ", 连接数量=", len(connList))
		}
	}
}

func (c *ConnectionManager) GetAllConnections() []Connection {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	conns := []Connection{}
	for _, value := range c.connections {
		conns = append(conns, value...)
	}
	return conns
}

func (c *ConnectionManager) GetConnectionGroup(connGroupId string) []Connection {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	connList, ok := c.connections[connGroupId]
	if !ok {
		return []Connection{}
	}
	return connList
}

func (c *ConnectionManager) GetConnection(connId string) Connection {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	for _, value := range c.connections {
		for _, connection := range value {
			if connId == connection.Id() {
				return connection
			}
		}
	}

	return nil
}
