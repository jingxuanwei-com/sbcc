# WebSocket 模块 — 使用文档

## 概述

WebSocket 模块提供实时双向通信能力，挂载在 Chi 路由底座上，与主服务共用 **9081 端口**。

## 设计思路

所有模块**共用一条 WebSocket 连接**，通过 `type` 字段区分消息来源：

```json
{"type":"home.random",   "payload":{"value":42}}
{"type":"podman.status", "payload":{"running":3}}
```

前端只需要连一个 `/websocket`，收所有模块的消息。

## 目录结构

```
websocket/
├── run.go      # 模块入口，将 /websocket 挂载到 Chi 底座
├── hub.go      # 连接管理器（注册/注销/广播/统计 + 全局 GlobalHub）
├── client.go   # 客户端封装（读写协程/心跳保活）
└── README.md   # 本文件
```

## 快速开始

### 服务端（已有，无需额外操作）

模块已在 `main/run.go` 中注册，启动后自动生效：

```go
// main/run.go
websocket.Run()
```

启动日志：

```
✅ [WebSocket] 全局端点 /websocket 已注册
```

### 各模块推送消息

```go
// 任何模块里都可以这样发消息（无需关心谁在收）
websocket.GlobalHub.BroadcastToType("模块名.事件名", map[string]interface{}{
    "key": "value",
})
```

### 客户端连接

#### JavaScript（浏览器）

```javascript
// 连一条 WS，收所有模块的消息
const ws = new WebSocket('ws://localhost:9081/websocket/ws')

ws.onmessage = (event) => {
  const { type, payload } = JSON.parse(event.data)

  switch (type) {
    case 'home.random':
      console.log('Home 数据:', payload)
      break
    case 'podman.status':
      console.log('Podman 状态:', payload)
      break
  }
}
```

#### curl（快速测试）

```bash
# 需要安装 wscat
# npm install -g wscat
wscat -c ws://localhost:9081/websocket/ws
```

---

## 服务端 API

### 全局 Hub

```go
import "modbus/websocket"

// 直接使用全局 Hub，不用自己 new
hub := websocket.GlobalHub
```

### `hub.BroadcastToType(type string, payload interface{})`

**推荐方式。** 按类型广播，自动打包为 `{"type":"...","payload":{...}}` 格式：

```go
GlobalHub.BroadcastToType("home.random", map[string]interface{}{
    "time":  "14:30:00",
    "value": 42,
})
```

### `hub.Broadcast(msg []byte)`

**底层方式。** 直接发原始字节，需自己打包 JSON：

```go
hub.Broadcast([]byte(`{"type":"home.random","payload":{"value":42}}`))
```

### `hub.Count() int`

获取当前在线客户端数量：

```go
if websocket.GlobalHub.Count() == 0 {
    // 没人连，不发了
    return
}
```

### `HandleWebSocket(w, r, hub, connID)`

WebSocket 握手处理器。已在 `run.go` 中自动注册到 `/websocket/ws`，一般无需手动调用。

参数：
- `w` / `r` — HTTP 响应和请求
- `hub` — 要使用的 Hub 实例（通常传 `GlobalHub`）
- `connID` — 连接 ID（需先通过 `hub.AddClient()` 注册）

---

## 接收客户端消息（自定义处理）

如果模块需要处理客户端发来的消息，设置 `OnMessage` 回调：

```go
// 在模块的 Run() 或 init() 中设置
websocket.GlobalHub.OnMessage = func(client *websocket.Client, data []byte) {
    log.Printf("收到消息: %s", string(data))
    // 在这里处理收到的消息...
}
```

### 向指定客户端发送消息

```go
// 通过 ID 找到客户端并发消息
hub := websocket.GlobalHub
// 注: clients 是私有字段，可通过 Broadcast 发送给所有人
// 如需定点发送，可扩展 Hub 添加 SendTo(id, msg) 方法
```

---

## 心跳机制

- **Ping 间隔**: 54 秒（`pongWait` 的 90%）
- **Pong 超时**: 60 秒
- 客户端 60 秒无响应自动断开

无需客户端额外处理，标准 WebSocket 库自动响应 Ping/Pong。

---

## 架构说明

```
客户端 ──ws──> Chi (9081)
                ├── GET /     → home 模块
                ├── GET /ws   → WebSocket Hub ── 管理多个 Client
                │                  ├── Client A (ReadPump + WritePump)
                │                  ├── Client B (ReadPump + WritePump)
                │                  └── Client C (ReadPump + WritePump)
                └── ...其他路由
```

- 与 Chi 共用端口，自动继承 Chi 的中间件（Recoverer、Logger、限流）
- 每个客户端独立协程，互不阻塞
