package gowssocket

type Handler interface {
	Connected(conn *Connection)
	Read(conn *Connection, data WsData) error
	Disconnected(conn *Connection, err error)
	Error(conn *Connection, err any)
}
