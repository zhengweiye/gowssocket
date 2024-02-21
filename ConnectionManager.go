package gowssocket

import (
	"fmt"
	"sync"
)

type ConnectionManager struct {
	connections    map[string][]*Connection
	connectionLock sync.RWMutex
	closeChan      chan byte
	registerChan   chan *Connection
	unregisterChan chan *Connection
}

func newConnectionManager() *ConnectionManager {
	connectionManager := &ConnectionManager{
		connections:    make(map[string][]*Connection),
		closeChan:      make(chan byte),
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
			c.remove(conn)
		case <-c.closeChan:
			return
		}
	}
}

func (c *ConnectionManager) shutdown() {
	c.connections = nil
	close(c.closeChan) //TODO 必须先关闭这个
	close(c.registerChan)
	close(c.unregisterChan)
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
	fmt.Println("服务名[", conn.GetServerName(), "]", "[AddConn] groupId=", conn.GroupId, ", connId=", conn.ConnId, ", exist=", exist, ", 连接数量=", len(c.connections[conn.GroupId]))
}

func (c *ConnectionManager) remove(conn *Connection) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	for groupId, value := range c.connections {
		delIndex := -1
		for index, item := range value {
			if item.ConnId == conn.ConnId {
				delIndex = index
				break
			}
		}
		if delIndex > -1 {
			connList := append(value[:delIndex], value[delIndex+1:]...)
			c.connections[groupId] = connList
			fmt.Println("服务名[", conn.GetServerName(), "]", "[RemoveConn] groupId=", groupId, ", connId=", conn.ConnId, ", 连接数量=", len(connList))
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
