package gowssocket

import (
	"fmt"
	"sync"
)

type ConnectionManager struct {
	connections    map[string][]*Connection
	connectionLock sync.RWMutex
	registerChan   chan *Connection
	unregisterChan chan *Connection
}

func newConnectionManager() *ConnectionManager {
	connectionManager := &ConnectionManager{
		connections:    make(map[string][]*Connection),
		registerChan:   make(chan *Connection, 10),
		unregisterChan: make(chan *Connection, 10),
	}

	go connectionManager.listen()

	return connectionManager
}

func (c *ConnectionManager) listen() {
	for {
		select {
		case conn := <-c.registerChan:
			c.add(conn)
		case conn := <-c.unregisterChan:
			c.remove(conn.ConnId)
		}
	}
}

func (c *ConnectionManager) add(conn *Connection) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()
	connectionList, ok := c.connections[conn.GroupId]
	if !ok {
		connectionList = []*Connection{}
	}

	exist := false
	for _, connection := range connectionList {
		if connection.ConnId == conn.ConnId {
			exist = true
			break
		}
	}
	if !exist {
		c.connections[conn.GroupId] = append(connectionList, conn)
	}
	fmt.Println("[AddConn] groupId=", conn.GroupId, ", connId=", conn.ConnId, ", exist=", exist, ", 连接数量=", len(c.connections[conn.GroupId]))
}

func (c *ConnectionManager) remove(connId string) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	for groupId, value := range c.connections {
		delIndex := -1
		for index, conn := range value {
			if conn.ConnId == connId {
				delIndex = index
				break
			}
		}
		if delIndex > -1 {
			connList := append(value[:delIndex], value[delIndex+1:]...)
			c.connections[groupId] = connList
			fmt.Println("[RemoveConn] groupId=", groupId, ", connId=", connId, ", 连接数量=", len(connList))
		}
	}
}

func (c *ConnectionManager) GetAllConnections() []*Connection {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	conns := []*Connection{}
	for _, value := range c.connections {
		conns = append(conns, value...)
	}
	return conns
}

func (c *ConnectionManager) GetConnectionGroup(connGroupId string) []*Connection {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	connList, ok := c.connections[connGroupId]
	if !ok {
		return []*Connection{}
	}
	return connList
}

func (c *ConnectionManager) GetConnection(connId string) *Connection {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	for _, value := range c.connections {
		for _, connection := range value {
			if connId == connection.ConnId {
				return connection
			}
		}
	}

	return nil
}
