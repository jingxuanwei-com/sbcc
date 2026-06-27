package websocket

const sendBufferSize = 256

// Conn — WebSocket 连接抽象
type Conn struct {
	ID     string
	UserID string
	hub    *Hub
	send   chan []byte
}

func newConn(id, userID string, hub *Hub) *Conn {
	return &Conn{
		ID:     id,
		UserID: userID,
		hub:    hub,
		send:   make(chan []byte, sendBufferSize),
	}
}

// Send — 向此连接发送数据（非阻塞）
func (c *Conn) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}
