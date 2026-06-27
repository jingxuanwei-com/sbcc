# WebSocket 模块

## 服务端

```go
import "modbus/websocket"

// 全局广播
websocket.Send("all", map[string]any{
    "type": "home.random", "payload": map[string]any{"value": 42},
})

// 发送到指定连接
websocket.Send("a1b2c3d4e5f6a7b8", map[string]any{
    "type": "chat.msg", "text": "hello",
})

// 接收客户端消息
websocket.OnMessage = func(connID string, data []byte) {
    log.Printf("[%s] 收到: %s", connID, data)
}
```

## 前端

```javascript
const ws = new WebSocket('ws://localhost:9081/websocket')
ws.onmessage = (event) => {
  const { type, payload } = JSON.parse(event.data)
  // ...
}
ws.send(JSON.stringify({hello: "world"})) // 发送消息到服务端
```
