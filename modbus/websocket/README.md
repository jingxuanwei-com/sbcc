# WebSocket 模块

所有模块共用一条 WebSocket 连接（`/websocket`），通过 `type` 字段区分消息来源。

```json
{"type":"home.random",   "payload":{"value":42}}
{"type":"podman.status", "payload":{"running":3}}
```

## 目录结构

```
websocket/
├── run.go      # 路由注册：挂载 /websocket
├── hub.go      # 连接管理器（注册/注销/广播/统计）
├── handler.go  # HTTP → WS 升级握手
├── pump.go     # 读写泵 + 心跳保活
├── conn.go     # 连接抽象
└── README.md
```

## 服务端 API

### 广播消息（任何模块）

```go
import "modbus/websocket"

// 推荐：按类型广播，自动打包为 {"type":"...","payload":{...}}
websocket.GlobalHub.BroadcastToType("home.random", map[string]interface{}{
    "time":  "14:30:00",
    "value": 42,
})

// 底层：直接发原始字节
websocket.GlobalHub.Broadcast([]byte(`{"type":"home.random","payload":{"value":42}}`))
```

### 在线统计

```go
if websocket.GlobalHub.Count() == 0 {
    return // 没人连，跳过
}
```

### 点对点发送

```go
// 给指定连接
hub.SendToConn("connID", []byte("hello"))

// 给某个用户的所有连接
hub.SendToUser("user123", []byte("hello"))
```

### 接收客户端消息

```go
websocket.GlobalHub.OnMessage = func(conn *websocket.Conn, data []byte) {
    log.Printf("收到: %s", string(data))
}
```

## 前端连接

```javascript
const ws = new WebSocket('ws://localhost:9081/websocket')

ws.onmessage = (event) => {
  const { type, payload } = JSON.parse(event.data)
  switch (type) {
    case 'home.random':
      console.log('数值:', payload.value)
      break
  }
}
```

## 心跳

- Ping 间隔: 54s / Pong 超时: 60s
- 无需客户端处理，标准 WS 库自动响应
