# WebSocket 模块 (ws)

基于 `github.com/gorilla/websocket`。

## 服务端

```go
import "modbus/ws"

// ─── 全局广播 ─────────────────────────
ws.Send("all", map[string]any{
    "type": "home.random", "payload": map[string]any{"value": 42},
})

// ─── 发送到指定连接 ───────────────────
ws.Send("a1b2c3d4e5f6a7b8", map[string]any{
    "type": "chat.msg", "text": "hello",
})

// ─── 按标签发送 ───────────────────────
ws.AddTag("a1b2c3d4e5f6a7b8", "room:chat", "group:admin")
ws.Send("tag:room:chat", map[string]any{"type": "chat.broadcast", "text": "Hi all!"})

// ─── 标签管理 ─────────────────────────
ws.AddTag(connID, "room:chat", "group:admin")     // 添加标签（支持多个）
ws.RemoveTag(connID, "group:admin")                // 移除标签
ws.GetConnIDsByTag("room:chat")                    // 获取标签下的所有连接ID
ws.GetTags(connID)                                 // 获取连接的所有标签

// ─── 获取连接 IP ──────────────────────
ip := ws.GetIP("a1b2c3d4e5f6a7b8")          // 返回 "192.168.1.100"
ids := ws.GetConnIDsByIP("192.168.1.100")    // IP 反查所有连接ID

// ─── 统计查询 ─────────────────────────
n := ws.CountAll()                           // 总连接数
n := ws.CountByIP("192.168.1.100")           // 指定IP的连接数
n := ws.CountByTag("room:chat")              // 指定标签的连接数
all := ws.GetAllConnIDs()                    // 所有连接ID
ips := ws.GetAllIPs()                        // 所有不同IP
st := ws.GetStats()                          // 完整统计（总数/IP数/分布）
log.Printf("总数:%d IP数:%d", st.Total, st.IPCount)

// ─── 接收客户端消息 ───────────────────
ws.OnMessage = func(connID string, data []byte) {
    log.Printf("[%s] 收到: %s", connID, data)
}
```

## 前端

```javascript
const ws = new WebSocket('ws://localhost:9081/ws')
ws.onmessage = (event) => {
  const { type, payload } = JSON.parse(event.data)
  // ...
}
ws.send(JSON.stringify({hello: "world"})) // 发送消息到服务端
```
