// ============================================================
//
//	Home 数据推送 WebSocket — SBCC 控制中心
//	功能：通过 WebSocket 向客户端推送实时数据（示例：随机数）
//	挂载路径："/data"（见 run.go）
//
// ============================================================
package home

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"modbus/websocket"
)

// dataHub 数据推送 Hub，复用 websocket.Hub 基建
var dataHub = websocket.NewHub("Home-Data")

// init 启动数据推送协程（包初始化时自动运行）
func init() {
	go pushLoop()
}

// pushLoop 每 10 秒广播一个随机数
func pushLoop() {
	for {
		time.Sleep(10 * time.Second)

		if dataHub.Count() == 0 {
			continue // 没人连，不发
		}

		num := rand.Intn(1000)
		msg := []byte(fmt.Sprintf(`{"time":"%s","value":%d}`,
			time.Now().Format("15:04:05"), num))

		dataHub.Broadcast(msg)
		log.Printf("📊 [Home-Data] 推送随机数: %d（在线: %d）", num, dataHub.Count())
	}
}

// handleData 处理 WebSocket 握手（路由已注册到 /data）
func handleData(w http.ResponseWriter, r *http.Request) {
	dataHub.HandleWebSocket(w, r)
}
