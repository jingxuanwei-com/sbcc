package websocket

import (
	"fmt"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

var connCounter int64

// handleGlobalWS — /websocket 的 WS 升级入口
func handleGlobalWS(w http.ResponseWriter, r *http.Request) {
	connCounter++
	connID := fmt.Sprintf("ws-%d", connCounter)
	GlobalHub.AddClient("", connID)
	HandleWebSocket(w, r, GlobalHub, connID)
}

// HandleWebSocket — HTTP 升 WS: 从 hub 取出 Conn 并挂载读写泵
func HandleWebSocket(w http.ResponseWriter, r *http.Request, hub *Hub, connID string) {
	conn := hub.GetConn(connID)
	if conn == nil {
		http.Error(w, "unknown conn", http.StatusNotFound)
		return
	}

	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		log.Printf("❌ %s WS 升级失败: %v", hub.logTag(), err)
		return
	}

	log.Printf("🔗 %s WS 挂载: %s（用户: %s）", hub.logTag(), connID, conn.UserID)
	go writePump(conn, ws)
	readPump(conn, ws, hub)
}
