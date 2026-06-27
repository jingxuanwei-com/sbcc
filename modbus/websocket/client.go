// ============================================================
//
//	WebSocket 客户端封装 — SBCC 控制中心
//	功能：管理单个 WebSocket 连接（读写、心跳、关闭）
//	依赖：github.com/coder/websocket
//
// ============================================================
package websocket

import (
	"context"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════════
//
//	常量配置
//
// ═══════════════════════════════════════════════════════════════
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufferSize = 256
)

// ═══════════════════════════════════════════════════════════════
//
//	Client — 一个 WebSocket 客户端连接
//
// ═══════════════════════════════════════════════════════════════
type Client struct {
	ID       string          // 客户端唯一标识
	UserID   int64           // 登录用户 ID（0=未登录）
	Username string          // 登录用户名
	conn     *websocket.Conn // WebSocket 连接
	hub      *Hub            // 所属 Hub
	send     chan []byte     // 消息通道
}

// NewClient — 创建客户端。clientID 为空则自动生成 UUID。
func NewClient(conn *websocket.Conn, hub *Hub, clientID string, userID int64, username string) *Client {
	if clientID == "" {
		clientID = uuid.New().String()
	}
	return &Client{
		ID:       clientID,
		UserID:   userID,
		Username: username,
		conn:     conn,
		hub:      hub,
		send:     make(chan []byte, sendBufferSize),
	}
}

// ReadPump — 读取泵：收消息
func (c *Client) ReadPump() {
	defer c.hub.Unregister(c)

	for {
		_, msg, err := c.conn.Read(context.Background())
		if err != nil {
			return
		}
		if c.hub.OnMessage != nil {
			c.hub.OnMessage(c, msg)
		}
	}
}

// WritePump — 写入泵：发消息 + 心跳
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer c.conn.Close(websocket.StatusNormalClosure, "closing")

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "closing")
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			c.conn.Write(ctx, websocket.MessageText, message)
			cancel()

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			c.conn.Ping(ctx)
			cancel()
		}
	}
}
