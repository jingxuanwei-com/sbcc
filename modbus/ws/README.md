# WebSocket 模块 (ws)

基于 `github.com/gorilla/websocket`。

## 服务端

```go
import "modbus/ws"

// 全局广播
ws.Send("all", map[string]any{
    "type": "home.random", "payload": map[string]any{"value": 42},
})

// 发送到指定连接
ws.Send("a1b2c3d4e5f6a7b8", map[string]any{
    "type": "chat.msg", "text": "hello",
})

// 接收客户端消息
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
