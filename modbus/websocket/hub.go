// ============================================================
//
//	WebSocket Hub 连接管理器 — SBCC 控制中心
//	功能：管理所有客户端，支持按用户/按 ID 发消息
//	依赖：github.com/coder/websocket
//
// ============================================================
package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

// ═══════════════════════════════════════════════════════════════
//
//	Hub — WebSocket 客户端连接管理器
//
// ═══════════════════════════════════════════════════════════════
type Hub struct {
	Name      string
	OnMessage func(client *Client, data []byte)

	mu      sync.RWMutex
	clients map[string]*Client        // clientID → Client
	users   map[int64]map[string]bool // userID → set of clientIDs
}

// GlobalHub — 全局统一的 WebSocket Hub，所有模块共用
var GlobalHub = NewHub("Global")

// Message — 标准 WebSocket 消息结构
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NewHub — 创建 Hub
func NewHub(name string) *Hub {
	return &Hub{
		Name:    name,
		clients: make(map[string]*Client),
		users:   make(map[int64]map[string]bool),
	}
}

func (h *Hub) logTag() string {
	if h.Name == "" {
		return "[WebSocket]"
	}
	return "[WebSocket-" + h.Name + "]"
}

// Register — 注册客户端。如果 clientID 已存在则替换旧连接。
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if old, ok := h.clients[client.ID]; ok {
		log.Printf("♻️ %s 客户端 %s 重复连接，替换旧连接", h.logTag(), client.ID)
		close(old.send)
		delete(h.clients, client.ID)
		if old.UserID > 0 {
			delete(h.users[old.UserID], old.ID)
			if len(h.users[old.UserID]) == 0 {
				delete(h.users, old.UserID)
			}
		}
	}

	h.clients[client.ID] = client
	if client.UserID > 0 {
		if h.users[client.UserID] == nil {
			h.users[client.UserID] = make(map[string]bool)
		}
		h.users[client.UserID][client.ID] = true
	}

	log.Printf("🔗 %s 客户端已连接: %s（用户: %s, 在线: %d）",
		h.logTag(), client.ID, client.Username, len(h.clients))
}

// Unregister — 注销客户端
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		close(client.send)
		delete(h.clients, client.ID)
		if client.UserID > 0 {
			delete(h.users[client.UserID], client.ID)
			if len(h.users[client.UserID]) == 0 {
				delete(h.users, client.UserID)
			}
		}
		log.Printf("🔌 %s 客户端已断开: %s（在线: %d）", h.logTag(), client.ID, len(h.clients))
	}
}

// Broadcast — 广播给所有客户端
func (h *Hub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.send <- data:
		default:
			log.Printf("⚠️ %s 客户端 %s 缓冲区满，丢消息", h.logTag(), client.ID)
		}
	}
}

// BroadcastToType — 格式化后广播
func (h *Hub) BroadcastToType(msgType string, payload interface{}) {
	msg, err := json.Marshal(Message{Type: msgType, Payload: payload})
	if err != nil {
		log.Printf("❌ %s 序列化消息失败: %v", h.logTag(), err)
		return
	}
	h.Broadcast(msg)
}

// SendToClient — 给指定 clientID 发消息。返回是否发送成功。
func (h *Hub) SendToClient(clientID string, data []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, ok := h.clients[clientID]
	if !ok {
		return false
	}
	select {
	case client.send <- data:
		return true
	default:
		log.Printf("⚠️ %s 发送给 %s 失败：缓冲区满", h.logTag(), clientID)
		return false
	}
}

// SendToClientType — 格式化后发给指定 clientID
func (h *Hub) SendToClientType(clientID string, msgType string, payload interface{}) bool {
	msg, err := json.Marshal(Message{Type: msgType, Payload: payload})
	if err != nil {
		return false
	}
	return h.SendToClient(clientID, msg)
}

// SendToUser — 给指定用户的所有连接发消息
func (h *Hub) SendToUser(userID int64, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clientIDs, ok := h.users[userID]
	if !ok {
		return
	}
	for cid := range clientIDs {
		if client, ok := h.clients[cid]; ok {
			select {
			case client.send <- data:
			default:
				log.Printf("⚠️ %s 发送给用户 %d(%s) 失败：缓冲区满", h.logTag(), userID, cid)
			}
		}
	}
}

// SendToUserType — 格式化后发给指定用户
func (h *Hub) SendToUserType(userID int64, msgType string, payload interface{}) {
	msg, err := json.Marshal(Message{Type: msgType, Payload: payload})
	if err != nil {
		return
	}
	h.SendToUser(userID, msg)
}

// Count — 在线客户端数
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// HandleWebSocket — WebSocket 握手入口
// 需通过 handleWS（run.go）调用，会从 context 中读取用户信息。
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, userID int64, username string) {
	clientID := r.URL.Query().Get("client_id")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("❌ %s 升级失败: %v", h.logTag(), err)
		return
	}

	client := NewClient(conn, h, clientID, userID, username)
	h.Register(client)

	go client.WritePump()
	client.ReadPump()
}
