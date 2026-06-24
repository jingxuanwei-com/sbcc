// ============================================================
//
//	Home 数据推送 — SBCC 控制中心
//	功能：通过统一 WebSocket 向客户端推送实时数据（示例：随机数）
//	消息类型：type = "home.random"，payload 含 time / value
//
// ============================================================
package home

import (
	"log"
	"math/rand" // 生成随机数作为示例数据
	"time"

	"modbus/websocket" // 统一 WebSocket 模块，提供 GlobalHub
)

// ═══════════════════════════════════════════════════════════════
//
//	init — 包初始化自启动
//	⚠️ Go 规则：一个包被 import 时，它的所有 init() 自动执行（只一次）
//	⚠️ 只要 main/run.go 导入了 home 包，这里就自动跑起来
//	⚠️ 无需任何人手动调用，也无所谓有没有调 home.Run()
//
// ═══════════════════════════════════════════════════════════════
func init() {
	// go 关键字启动一个 goroutine（轻量级协程）
	// 这意味着 pushLoop 在后台并发运行，不会阻塞主流程
	go pushLoop()
}

// ═══════════════════════════════════════════════════════════════
//
//	pushLoop — 后台数据推送循环
//	工作机制：
//	  1. 每 10 秒执行一轮
//	  2. 先检查有没有 WS 客户端连着，没人连就跳过
//	  3. 生成随机数 → 包装成标准消息格式 → 通过 GlobalHub 广播
//	这是一种"生产者-消费者"模式：
//	  生产者: 这个循环 (pushLoop)
//	  消费者: 所有前端 WS 客户端
//	  通道:    GlobalHub → Hub.Broadcast → client.Send chan
//
// ═══════════════════════════════════════════════════════════════
func pushLoop() {
	// for {} 是 Go 的死循环，等价于 while(true)
	for {
		// ─── 1. 定时：每 10 秒一轮 ─────────────────────────
		time.Sleep(10 * time.Second)

		// ─── 2. 检查是否有 WS 客户端在线 ────────────────────
		// 如果没人连 WS，推送了也没人收，直接跳过本轮
		if websocket.GlobalHub.Count() == 0 {
			continue // 跳过当前循环剩余代码，进入下一轮
		}

		// ─── 3. 生成示例数据 ────────────────────────────────
		// rand.Intn(1000) 生成 0-999 之间的随机整数
		// 实际项目中这里会替换为真实数据（如传感器读数、设备状态等）
		num := rand.Intn(1000)

		// ─── 4. 广播给所有连接的客户端 ─────────────────────
		// BroadcastToType 会自动:
		//   a) 将数据序列化为 JSON → {"type":"home.random","payload":{...}}
		//   b) 通过 GlobalHub 发给所有连了 WS 的客户端
		//
		// 消息协议：
		//   type:    "模块名.事件名" 点号分隔，前端用 switch 区分
		//   payload: 实际数据，map 或 struct 皆可
		websocket.GlobalHub.BroadcastToType("home.random", map[string]interface{}{
			"time":  time.Now().Format("15:04:05"), // 当前时间（时:分:秒）
			"value": num,                           // 随机数值
		})

		// ─── 5. 服务端日志 ─────────────────────────────────
		log.Printf("📊 [Home] 推送随机数: %d（在线: %d）", num, websocket.GlobalHub.Count())
	}
}
