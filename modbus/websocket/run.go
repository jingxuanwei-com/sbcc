// ============================================================
//
//	WebSocket 服务模块 — SBCC 控制中心
//	功能：管理实时双向通信连接，挂载到 Chi 路由底座
//	依赖：github.com/gorilla/websocket
//	启动方式：由 modbus/main 模块统一调用
//
// ============================================================
package websocket

import (
	"log" // 日志输出

	web "modbus/chi" // Chi 路由底座（复用端口 9081）
	"modbus/env"     // 环境配置（读取 WEBSOCKET_PATH）
)

// Hub 全局实例（包级变量）
// 所有 WebSocket 连接管理通过此实例进行
var GlobalHub *Hub

// Run 初始化 WebSocket 服务
// 步骤：
//  1. 初始化配置（WEBSOCKET_PATH）
//  2. 创建全局 Hub 连接管理器
//  3. 将 WebSocket 端点挂载到 Chi 路由底座
//
// 注意：与 chi 共用端口 9081，无需单独开端口
func Run() {
	// 第一步：初始化 WebSocket 配置
	env.Init([][]string{
		{"WEBSOCKET_PATH", "/ws", "WebSocket 连接路径"},
	})

	// 第二步：从配置读取路径
	wsPath := env.Get("WEBSOCKET_PATH")

	// 第三步：创建全局 Hub 连接管理器
	GlobalHub = NewHub()

	// 第四步：将 WebSocket 握手端点注册到 Chi 底座
	// 直接注册在 chi 的主路由上，与 home 模块互不冲突
	web.Mux.Get(wsPath, GlobalHub.HandleWebSocket)

	log.Printf("✅ [WebSocket] 已挂载到 Chi 底座，端点: %s", wsPath)
}
