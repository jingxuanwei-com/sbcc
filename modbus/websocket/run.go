// ============================================================
//
//	WebSocket 模块启动 — SBCC 控制中心
//	功能：注册 HTTP → WebSocket 升级端点
//
//	服务器通过 hub.AddClient() 注册连接，
//	HandleWebSocket 将真实 WS 传输挂载到已有连接上面。
//
// ============================================================
package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	web "modbus/chi"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
)

var connCounter int64

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

func Run() {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "use ws://", http.StatusUpgradeRequired)
	})

	// WebSocket 升级端点
	r.Get("/ws", handleGlobalWS)

	web.Mux.Mount("/websocket", r)
	log.Print("✅ [WebSocket] 端点 /websocket 已注册")
}

// handleGlobalWS — 全局 WebSocket 升级处理器
// 客户端连接 ws://host:port/websocket/ws 即可接收所有模块的广播
func handleGlobalWS(w http.ResponseWriter, r *http.Request) {
	connCounter++
	connID := fmt.Sprintf("ws-%d", connCounter)

	hub := GlobalHub
	hub.AddClient("", connID)
	HandleWebSocket(w, r, hub, connID)
}

// HandleWebSocket — 将已有连接挂载到 WebSocket 传输层
//
//	connID 必须已通过 hub.AddClient() 注册。
//	调用后，该连接将开始接收和发送真实 WS 数据。
//
//	用法：
//	  hub.AddClient("user123", "connA")
//	  // ... 在 HTTP handler 中：
//	  ws.HandleWebSocket(w, r, hub, "connA")
func HandleWebSocket(w http.ResponseWriter, r *http.Request, hub *Hub, connID string) {
	conn := hub.GetConn(connID)
	if conn == nil {
		http.Error(w, "unknown conn", http.StatusNotFound)
		return
	}

	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("❌ %s WS 升级失败: %v", hub.logTag(), err)
		return
	}

	log.Printf("🔗 %s WS 传输层挂载: %s（用户: %s）", hub.logTag(), connID, conn.UserID)

	// 启动写泵
	go writePump(conn, ws)

	// 读泵（阻塞直到断开）
	readPump(conn, ws, hub)
}

// writePump — 从 conn.send 读到数据后写入 WS
func writePump(conn *Conn, ws *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer ws.Close(websocket.StatusNormalClosure, "closing")

	for {
		select {
		case msg, ok := <-conn.send:
			if !ok {
				ws.Close(websocket.StatusNormalClosure, "closing")
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			ws.Write(ctx, websocket.MessageText, msg)
			cancel()

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			ws.Ping(ctx)
			cancel()
		}
	}
}

// readPump — 从 WS 读到数据后回调 hub.OnMessage
func readPump(conn *Conn, ws *websocket.Conn, hub *Hub) {
	defer func() {
		ws.Close(websocket.StatusNormalClosure, "closed")
		hub.RemoveConn(conn.ID)
	}()

	for {
		_, msg, err := ws.Read(context.Background())
		if err != nil {
			return
		}
		if hub.OnMessage != nil {
			hub.OnMessage(conn, msg)
		}
	}
}
