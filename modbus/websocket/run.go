// ============================================================
//
//	WebSocket 模块启动 — SBCC 控制中心
//	功能：注册全局 WebSocket 统一端点，供所有模块共用
//	挂载路径："/websocket"
//
// ============================================================
package websocket

import (
	"log"

	web "modbus/chi"

	"github.com/go-chi/chi/v5"
)

// Run 注册全局 WebSocket 端点
// 所有模块共享 GlobalHub，前端只需连一条 WS
func Run() {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Get("/", GlobalHub.HandleWebSocket)
	})

	web.Mux.Mount("/websocket", r)
	log.Print("✅ [WebSocket] 全局端点 /websocket 已注册")
}

