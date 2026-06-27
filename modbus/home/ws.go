package home

import (
	"log"
	"math/rand"
	"time"

	"modbus/websocket"
)

func init() {
	go pushLoop()

	websocket.OnMessage = func(connID string, data []byte) {
		log.Printf("[%s] 收到: %s", connID, data)
	}
}

func pushLoop() {
	for {
		time.Sleep(10 * time.Second)

		num := rand.Intn(1000)

		websocket.Send("all", map[string]any{
			"type":    "home.random",
			"payload": map[string]any{"time": time.Now().Format("15:04:05"), "value": num},
		})

		log.Printf("📊 [Home] 推送随机数: %d", num)
	}
}
