package ws

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	mu       sync.Mutex
	conns    = map[string]*websocket.Conn{}
	tagConns = map[string]map[string]struct{}{} // tag → set of connIDs
	connTags = map[string]map[string]struct{}{} // connID → set of tags
)

// OnMessage 收到客户端消息时的回调，connID 是发送者的连接 ID
var OnMessage func(connID string, data []byte)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

func randID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ─── 标签管理 ─────────────────────────────────────────

// AddTag 为指定连接添加一个或多个标签。
func AddTag(connID string, tags ...string) {
	mu.Lock()
	defer mu.Unlock()
	for _, tag := range tags {
		if tagConns[tag] == nil {
			tagConns[tag] = map[string]struct{}{}
		}
		tagConns[tag][connID] = struct{}{}

		if connTags[connID] == nil {
			connTags[connID] = map[string]struct{}{}
		}
		connTags[connID][tag] = struct{}{}
	}
}

// RemoveTag 移除指定连接的指定标签。
func RemoveTag(connID string, tags ...string) {
	mu.Lock()
	defer mu.Unlock()
	for _, tag := range tags {
		if set := tagConns[tag]; set != nil {
			delete(set, connID)
			if len(set) == 0 {
				delete(tagConns, tag)
			}
		}
		if set := connTags[connID]; set != nil {
			delete(set, tag)
		}
	}
}

// GetConnIDsByTag 返回拥有指定标签的所有连接 ID。
func GetConnIDsByTag(tag string) []string {
	mu.Lock()
	defer mu.Unlock()
	set := tagConns[tag]
	if set == nil {
		return nil
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}

// GetTags 返回指定连接的所有标签。
func GetTags(connID string) []string {
	mu.Lock()
	defer mu.Unlock()
	set := connTags[connID]
	if set == nil {
		return nil
	}
	tags := make([]string, 0, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	return tags
}

// removeConnTags 移除指定连接的所有标签（内部调用，不锁）。
func removeConnTags(connID string) {
	if set := connTags[connID]; set != nil {
		for tag := range set {
			if s := tagConns[tag]; s != nil {
				delete(s, connID)
				if len(s) == 0 {
					delete(tagConns, tag)
				}
			}
		}
		delete(connTags, connID)
	}
}

// ─── 发送消息 ─────────────────────────────────────────

// Send 发送消息。
//   - id="all" 时全局广播
//   - id 以 "tag:" 开头（如 "tag:room:chat"）时，发送给该标签的所有连接
//   - 否则发给指定 connID
func Send(id string, data map[string]any) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	switch {
	case id == "all":
		for cid, c := range conns {
			c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if e := c.WriteMessage(websocket.TextMessage, msg); e != nil {
				delete(conns, cid)
				removeConnTags(cid)
			}
		}

	case len(id) > 4 && id[:4] == "tag:":
		tag := id[4:]
		for cid := range tagConns[tag] {
			if c := conns[cid]; c != nil {
				c.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if e := c.WriteMessage(websocket.TextMessage, msg); e != nil {
					delete(conns, cid)
					removeConnTags(cid)
				}
			}
		}

	default:
		if c := conns[id]; c != nil {
			c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			c.WriteMessage(websocket.TextMessage, msg)
		}
	}
}

// Handler /ws 的 HTTP 处理器
func Handler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ [WS] 升级失败: %v", err)
		return
	}

	mu.Lock()
	id := randID()
	conns[id] = c
	n := len(conns)
	mu.Unlock()
	log.Printf("🔗 [WS] %s（在线: %d）", id, n)

	defer func() {
		c.Close()
		mu.Lock()
		delete(conns, id)
		removeConnTags(id) // 断开时自动清除所有标签
		n := len(conns)
		mu.Unlock()
		log.Printf("🔌 [WS] %s（在线: %d）", id, n)
	}()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		if OnMessage != nil {
			OnMessage(id, msg)
		}
	}
}
