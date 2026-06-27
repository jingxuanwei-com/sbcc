package websocket

import (
	"context"
	"time"

	"github.com/coder/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// writePump — 从 conn.send 读出消息写入 WS 连接，同时定时发 Ping
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

// readPump — 从 WS 读取消息，回调 hub.OnMessage；断开时自动清理
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
