package websocket

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub — WebSocket 消息路由器
type Hub struct {
	Name      string
	OnMessage func(conn *Conn, data []byte)

	mu    sync.RWMutex
	conns map[string]*Conn
	users map[string]map[string]bool
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

// Count — 在线连接数
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}
