package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

var (
	mu    sync.Mutex
	conns = map[string]*websocket.Conn{}
)

// OnMessage 收到客户端消息时的回调，connID 是发送者的连接 ID
var OnMessage func(connID string, data []byte)

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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if e := c.Write(ctx, websocket.MessageText, msg); e != nil {
				delete(conns, cid)
			}
			cancel()
		}
	} else if c := conns[id]; c != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c.Write(ctx, websocket.MessageText, msg)
		cancel()
	}
}

// Handler /websocket 的 HTTP 处理器
func Handler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
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
		c.CloseNow()
		mu.Lock()
		delete(conns, id)
		n := len(conns)
		mu.Unlock()
		log.Printf("🔌 [WS] %s（在线: %d）", id, n)
	}()

	for {
		_, msg, err := c.Read(context.Background())
		if err != nil {
			return
		}
		if OnMessage != nil {
			OnMessage(id, msg)
		}
	}
}
