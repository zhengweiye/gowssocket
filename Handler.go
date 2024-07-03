package gowssocket

type Handler interface {
	Connected(conn Connection)
	Read(conn Connection, data HandlerData) error
	Disconnected(conn Connection)
	Error(conn Connection, err any)
}

type HandlerData struct {
	MessageType int    `json:"messageType"`
	MessageData []byte `json:"messageData"`
}
