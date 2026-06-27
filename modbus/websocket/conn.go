// ============================================================
//
//	WebSocket 连接封装 — SBCC 控制中心
//	功能：统一连接抽象，支持 WS 和编程式客户端
//
// ============================================================
package websocket

const sendBufferSize = 256

// Conn — 一个连接抽象
// 可以是 WebSocket 连接，也可以是纯编程式连接。
type Conn struct {
	ID     string      // 连接 ID（服务端生成）
	UserID string      // 用户 ID（字符串，灵活适应各种场景）
	hub    *Hub        // 所属 Hub
	send   chan []byte // 消息通道
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

// Recv — 阻塞接收消息（编程式客户端用）
func (c *Conn) Recv() []byte {
	return <-c.send
}

// C — 消息通道，供 select 使用
// 用法：
//
//	select {
//	case msg := <-conn.C():
//	    // 处理消息
//	case <-time.After(5*time.Second):
//	    // 超时
//	}
func (c *Conn) C() <-chan []byte {
	return c.send
}
