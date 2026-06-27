// ============================================================
//
//	WebSocket 模块启动 — SBCC 控制中心
//	功能：注册全局 WebSocket 端点，支持登录认证
//	依赖：github.com/coder/websocket、scs
//	挂载路径："/websocket"
//
//	端点：
//	  ws://host/websocket               — 自动从 session 读取登录信息
//	  ws://host/websocket?client_id=xxx — 指定客户端 ID
//
//	认证：
//	  必须先登录获取 session cookie，未登录返回 401。
//
// ============================================================
package websocket

import (
	"log"
	"net/http"

	web "modbus/chi"
	"modbus/scs"

	"github.com/go-chi/chi/v5"
)

// ═══════════════════════════════════════════════════════════════
//
//	Run — WebSocket 模块启动入口
//
// ═══════════════════════════════════════════════════════════════
func Run() {
	r := chi.NewRouter()

	// 包裹 Session 中间件，从 cookie 加载 session
	r.Group(func(r chi.Router) {
		r.Use(scs.Scs.LoadAndSave)
		r.Get("/", handleWS)
	})

	web.Mux.Mount("/websocket", r)
	log.Print("✅ [WebSocket] 全局端点 /websocket 已注册（需登录）")
}

// handleWS 检查登录，然后传给 Hub 处理 WebSocket 握手
func handleWS(w http.ResponseWriter, r *http.Request) {
	userID := scs.Scs.GetInt64(r.Context(), "user_id")
	username := scs.Scs.GetString(r.Context(), "username")
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	GlobalHub.HandleWebSocket(w, r, userID, username)
}
