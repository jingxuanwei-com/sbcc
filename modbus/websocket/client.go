// ============================================================
//
//	WebSocket 客户端封装 — SBCC 控制中心
//	功能：管理单个 WebSocket 连接的生命周期（读写、心跳、关闭）
//	依赖：gorilla/websocket
//	启动方式：由 Hub.HandleWebSocket 自动创建
//
// ============================================================
package websocket

import (
	"log"  // 日志输出
	"time" // 时间控制（心跳、超时）

	"github.com/google/uuid"       // UUID 生成（客户端 ID）
	"github.com/gorilla/websocket" // WebSocket 库
)

const (
	// 写入等待超时（向客户端发送消息的最长等待时间）
	writeWait = 10 * time.Second

	// 支持的消息类型（pong 响应）
	pongWait = 60 * time.Second

	// 心跳发送间隔（必须小于 pongWait）
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小（字节）
	maxMessageSize = 4096

	// 发送缓冲区大小
	sendBufferSize = 256
)

// Client 代表一个 WebSocket 客户端连接
type Client struct {
	// 客户端唯一标识
	ID string

	// WebSocket 连接对象
	Conn *websocket.Conn

	// 所属的 Hub 连接管理器
	Hub *Hub

	// 发送消息的缓冲通道
	Send chan []byte
}

// NewClient 创建一个新的客户端实例
func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:   uuid.New().String(),
		Conn: conn,
		Hub:  hub,
		Send: make(chan []byte, sendBufferSize),
	}
}

// ReadPump 读取协程：从 WebSocket 连接读取消息
// 持续读取客户端发来的消息，并在连接断开时清理资源
func (c *Client) ReadPump() {
	defer func() {
		// 客户端断开连接时，从 Hub 中注销
		c.Hub.Unregister(c.ID)
	}()

	// 设置读取参数
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
				websocket.CloseAbnormalClosure) {
				log.Printf("⚠️ [WebSocket] 客户端 %s 异常断开: %v", c.ID, err)
			}
			break
		}

		if c.Hub.OnMessage != nil {
			// 调用模块自定义的消息处理
			c.Hub.OnMessage(c, message)
		} else {
			log.Printf("📩 %s 收到来自 %s 的消息: %s", c.Hub.logTag(), c.ID, string(message))
		}
	}
}

// WritePump 写入协程：向 WebSocket 连接发送消息
// 从 Send 通道读取消息，并写入 WebSocket 连接
// 同时定期发送 Ping 帧以维持心跳
func (c *Client) WritePump() {
	// 创建一个定时器，用于定期发送 Ping
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			// 设置写入超时
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 通道已关闭，发送关闭帧
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 获取写入器
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 将缓冲队列中剩余的消息一并写入（减少系统调用）
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// 定期发送 Ping 心跳包
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
