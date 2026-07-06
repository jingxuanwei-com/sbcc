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
	mu    sync.Mutex
	conns = map[string]*websocket.Conn{}
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

// Send 发送消息。id="all" 时全局广播，否则发给指定连接。
func Send(id string, data map[string]any) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if id == "all" {
		for cid, c := range conns {
			c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if e := c.WriteMessage(websocket.TextMessage, msg); e != nil {
				delete(conns, cid)
			}
		}
	} else if c := conns[id]; c != nil {
		c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		c.WriteMessage(websocket.TextMessage, msg)
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
