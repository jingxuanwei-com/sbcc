// ============================================================
//
//	WebSocket Hub — SBCC 控制中心
//	功能：纯消息路由，支持按用户/按连接发消息
//
//	用法：
//	  hub := ws.NewHub()
//
//	  // 添加客户端（由服务端注册）
//	  hub.AddClient("user123", "connA")
//	  hub.AddClient("user123", "connB")
//
//	  // 全局广播
//	  hub.Broadcast([]byte("系统公告"))
//
//	  // 给某个用户所有连接发
//	  hub.SendToUser("user123", []byte("你好"))
//
//	  // 给某个连接单独发
//	  hub.SendToConn("connA", []byte("只给 connA"))
//
// ============================================================
package websocket

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub — 消息路由器
type Hub struct {
	Name      string
	OnMessage func(conn *Conn, data []byte) // 收到消息回调

	mu    sync.RWMutex
	conns map[string]*Conn           // connID → Conn
	users map[string]map[string]bool // userID → set of connIDs
}

// GlobalHub — 全局默认 Hub
var GlobalHub = NewHub("Global")

// Message — 标准消息结构
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NewHub — 创建 Hub
func NewHub(name string) *Hub {
	return &Hub{
		Name:  name,
		conns: make(map[string]*Conn),
		users: make(map[string]map[string]bool),
	}
}

func (h *Hub) logTag() string {
	if h.Name == "" {
		return "[WS]"
	}
	return "[WS-" + h.Name + "]"
}

// AddClient — 添加客户端。如果 connID 已存在则替换。
func (h *Hub) AddClient(userID, connID string) *Conn {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 替换旧连接
	if old, ok := h.conns[connID]; ok {
		close(old.send)
		delete(h.conns, connID)
		if old.UserID != "" {
			delete(h.users[old.UserID], connID)
			if len(h.users[old.UserID]) == 0 {
				delete(h.users, old.UserID)
			}
		}
	}

	conn := newConn(connID, userID, h)
	h.conns[connID] = conn

	if userID != "" {
		if h.users[userID] == nil {
			h.users[userID] = make(map[string]bool)
		}
		h.users[userID][connID] = true
	}

	log.Printf("🔗 %s 添加连接: %s（用户: %s, 在线: %d）",
		h.logTag(), connID, userID, len(h.conns))

	return conn
}

// RemoveConn — 移除连接
func (h *Hub) RemoveConn(connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, ok := h.conns[connID]; ok {
		close(conn.send)
		delete(h.conns, connID)
		if conn.UserID != "" {
			delete(h.users[conn.UserID], connID)
			if len(h.users[conn.UserID]) == 0 {
				delete(h.users, conn.UserID)
			}
		}
		log.Printf("🔌 %s 移除连接: %s（在线: %d）", h.logTag(), connID, len(h.conns))
	}
}

// GetConn — 获取连接
func (h *Hub) GetConn(connID string) *Conn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.conns[connID]
}

// Broadcast — 广播给所有连接
func (h *Hub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, conn := range h.conns {
		conn.Send(data)
	}
}

// BroadcastToType — 格式化后广播
func (h *Hub) BroadcastToType(msgType string, payload interface{}) {
	msg, err := json.Marshal(Message{Type: msgType, Payload: payload})
	if err != nil {
		return
	}
	h.Broadcast(msg)
}

// SendToConn — 给指定连接发消息
func (h *Hub) SendToConn(connID string, data []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conn, ok := h.conns[connID]
	if !ok {
		return false
	}
	return conn.Send(data)
}

// SendToConnType — 格式化后发给指定连接
func (h *Hub) SendToConnType(connID string, msgType string, payload interface{}) bool {
	msg, err := json.Marshal(Message{Type: msgType, Payload: payload})
	if err != nil {
		return false
	}
	return h.SendToConn(connID, msg)
}

// SendToUser — 给用户所有连接发消息
func (h *Hub) SendToUser(userID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connIDs, ok := h.users[userID]
	if !ok {
		return
	}
	for cid := range connIDs {
		if conn, ok := h.conns[cid]; ok {
			conn.Send(data)
		}
	}
}

// SendToUserType — 格式化后发给用户
func (h *Hub) SendToUserType(userID string, msgType string, payload interface{}) {
	msg, err := json.Marshal(Message{Type: msgType, Payload: payload})
	if err != nil {
		return
	}
	h.SendToUser(userID, msg)
}

// Count — 在线连接数
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

// Conns — 返回所有连接 ID 列表
func (h *Hub) Conns() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.conns))
	for id := range h.conns {
		ids = append(ids, id)
	}
	return ids
}

// UserConns — 返回用户所有连接 ID
func (h *Hub) UserConns(userID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connIDs, ok := h.users[userID]
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(connIDs))
	for id := range connIDs {
		ids = append(ids, id)
	}
	return ids
}
