// ============================================================
//
//	WebSocket 模块启动 — SBCC 控制中心
//	功能：注册全局 WebSocket 统一端点，供所有模块共用
//	挂载路径："/websocket"
//
//	📖 这里在做什么？
//	在 Chi 路由底座上注册一个路径 /websocket，
//	用户浏览器访问 http://地址/websocket 时，
//	Chi 会自动调 GlobalHub.HandleWebSocket 完成握手。
//
// ============================================================
package websocket

import (
	"log"

	web "modbus/chi" // web = Chi 路由底座，所有模块都挂在这里

	"github.com/go-chi/chi/v5" // Chi 路由库
)

// ═══════════════════════════════════════════════════════════════
//
//	Run — WebSocket 模块启动入口
//
//	在 main/run.go 里调用：
//	  chi.Run()
//	  websocket.Run()   ← 在这里
//	  home.Run()
//	  podman.Run()
//
//	做了两件事：
//	  1. 创建子路由器 r
//	  2. 把 /websocket 挂到 Chi 底座上
//
//	效果：
//	  用户访问 http://localhost:9081/websocket → WebSocket 握手
//
// ═══════════════════════════════════════════════════════════════
func Run() {
	// ─── 1. 创建子路由器 ──────────────────────────────────────
	// chi.NewRouter() 创建一个小型路由器，有自己的路由表
	// 之后用 web.Mux.Mount 把它挂到主路由器上
	r := chi.NewRouter()

	// ─── 2. 注册路由 ──────────────────────────────────────────
	// r.Group 把一组路由打包在一起
	// 这里只有一条：访问 / 就处理 WebSocket 握手
	//
	// 注意：这里写的是 "/" 不是 "/websocket"
	// 因为 Mount("/websocket", r) 已经把 r 挂到了 /websocket 下
	// 所以 r 里的 "/" 实际对外就是 "/websocket"
	r.Group(func(r chi.Router) {
		r.Get("/", GlobalHub.HandleWebSocket)
	})

	// ─── 3. 挂载到 Chi 底座 ──────────────────────────────────
	// Mount 把 r 整个挂到 /websocket 路径下
	// 相当于告诉 Chi："访问 /websocket 的请求都交给 r 处理"
	web.Mux.Mount("/websocket", r)
	log.Print("✅ [WebSocket] 全局端点 /websocket 已注册")
}
