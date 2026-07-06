package ws

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// peer 包装一个 WebSocket 连接，自带写锁保证并发安全。
type peer struct {
	id   string
	conn *websocket.Conn
	wmu  sync.Mutex
}

var (
	mu       sync.Mutex
	peers    = map[string]*peer{}
	connIPs  = map[string]string{}              // connID → IP 地址
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

// realIP 从请求中提取真实客户端 IP，优先取 X-Forwarded-For / X-Real-IP。
func realIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if i := strings.Index(fwd, ","); i > 0 {
			return strings.TrimSpace(fwd[:i])
		}
		return strings.TrimSpace(fwd)
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return strings.TrimSpace(real)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// GetIP 返回指定连接 ID 的客户端 IP 地址。
func GetIP(connID string) string {
	mu.Lock()
	defer mu.Unlock()
	return connIPs[connID]
}

// GetConnIDsByIP 返回指定 IP 的所有连接 ID。
func GetConnIDsByIP(ip string) []string {
	mu.Lock()
	defer mu.Unlock()
	var ids []string
	for id, addr := range connIPs {
		if addr == ip {
			ids = append(ids, id)
		}
	}
	return ids
}

// CountAll 返回当前总连接数。
func CountAll() int {
	mu.Lock()
	defer mu.Unlock()
	return len(peers)
}

// CountByIP 返回指定 IP 的连接数。
func CountByIP(ip string) int {
	mu.Lock()
	defer mu.Unlock()
	n := 0
	for _, addr := range connIPs {
		if addr == ip {
			n++
		}
	}
	return n
}

// CountByTag 返回指定标签的连接数。
func CountByTag(tag string) int {
	mu.Lock()
	defer mu.Unlock()
	return len(tagConns[tag])
}

// GetAllConnIDs 返回所有连接 ID。
func GetAllConnIDs() []string {
	mu.Lock()
	defer mu.Unlock()
	ids := make([]string, 0, len(peers))
	for id := range peers {
		ids = append(ids, id)
	}
	return ids
}

// GetAllIPs 返回当前所有不同的客户端 IP。
func GetAllIPs() []string {
	mu.Lock()
	defer mu.Unlock()
	seen := map[string]struct{}{}
	for _, addr := range connIPs {
		seen[addr] = struct{}{}
	}
	ips := make([]string, 0, len(seen))
	for ip := range seen {
		ips = append(ips, ip)
	}
	return ips
}

// ConnStats 连接概要统计。
type ConnStats struct {
	Total   int            // 总连接数
	IPCount int            // 不同 IP 数量
	ByIP    map[string]int // 每个 IP 的连接数
	ByTag   map[string]int // 每个标签的连接数
}

func GetStats() *ConnStats {
	mu.Lock()
	defer mu.Unlock()

	s := &ConnStats{
		Total: len(peers),
		ByIP:  map[string]int{},
		ByTag: map[string]int{},
	}
	for _, addr := range connIPs {
		s.ByIP[addr]++
	}
	for tag, set := range tagConns {
		s.ByTag[tag] = len(set)
	}
	s.IPCount = len(s.ByIP)
	return s
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

// removeConnMeta 移除指定连接的所有元数据（IP、标签）（内部调用，不锁）。
func removeConnMeta(connID string) {
	delete(connIPs, connID)
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

	switch {
	case id == "all":
		// 快照目标列表，避免长时间持锁
		mu.Lock()
		targets := make([]*peer, 0, len(peers))
		for _, p := range peers {
			targets = append(targets, p)
		}
		mu.Unlock()

		for _, p := range targets {
			p.wmu.Lock()
			p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if e := p.conn.WriteMessage(websocket.TextMessage, msg); e != nil {
				p.wmu.Unlock()
				mu.Lock()
				delete(peers, p.id)
				removeConnMeta(p.id)
				mu.Unlock()
				continue
			}
			p.wmu.Unlock()
		}

	case len(id) > 4 && id[:4] == "tag:":
		tag := id[4:]
		mu.Lock()
		set := tagConns[tag]
		targets := make([]*peer, 0, len(set))
		for cid := range set {
			if p := peers[cid]; p != nil {
				targets = append(targets, p)
			}
		}
		mu.Unlock()

		for _, p := range targets {
			p.wmu.Lock()
			p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if e := p.conn.WriteMessage(websocket.TextMessage, msg); e != nil {
				p.wmu.Unlock()
				mu.Lock()
				delete(peers, p.id)
				removeConnMeta(p.id)
				mu.Unlock()
				continue
			}
			p.wmu.Unlock()
		}

	default:
		mu.Lock()
		p := peers[id]
		mu.Unlock()

		if p == nil {
			return
		}
		p.wmu.Lock()
		defer p.wmu.Unlock()
		p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		p.conn.WriteMessage(websocket.TextMessage, msg)
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
	p := &peer{id: id, conn: c}
	peers[id] = p
	connIPs[id] = realIP(r)
	n := len(peers)
	mu.Unlock()
	log.Printf("🔗 [WS] %s（在线: %d） [%s]", id, n, connIPs[id])

	defer func() {
		c.Close()
		mu.Lock()
		delete(peers, id)
		removeConnMeta(id)
		n := len(peers)
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
