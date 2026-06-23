# WebSocket 模块 — 使用文档

## 概述

WebSocket 模块提供实时双向通信能力，挂载在 Chi 路由底座上，与主服务共用 **9081 端口**。

## 目录结构

```
websocket/
├── run.go      # 模块入口，挂载到 Chi 底座
├── hub.go      # 连接管理器（注册/注销/广播/统计）
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
✅ [Chi] 9081端口占领成功，底座已就绪
✅ [WebSocket] 已挂载到 Chi 底座，端点: /ws
```

### 客户端连接

#### JavaScript（浏览器）

```javascript
const ws = new WebSocket('ws://localhost:9081/ws')

ws.onopen = () => {
  console.log('✅ 已连接')
  ws.send('Hello SBCC!')
}

ws.onmessage = (event) => {
  console.log('📩 收到:', event.data)
}

ws.onclose = () => console.log('🔌 已断开')
ws.onerror = (err) => console.error('❌ 错误:', err)
```

#### Python

```python
import asyncio
import websockets

async def connect():
    async with websockets.connect('ws://localhost:9081/ws') as ws:
        print('✅ 已连接')
        await ws.send('Hello SBCC!')

        async for message in ws:
            print(f'📩 收到: {message}')

asyncio.run(connect())
```

#### Go

```go
package main

import (
    "log"
    "net/url"

    "github.com/gorilla/websocket"
)

func main() {
    u := url.URL{Scheme: "ws", Host: "localhost:9081", Path: "/ws"}
    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    log.Println("✅ 已连接")

    // 发送
    conn.WriteMessage(websocket.TextMessage, []byte("Hello SBCC!"))

    // 接收
    _, msg, _ := conn.ReadMessage()
    log.Printf("📩 收到: %s", msg)
}
```

#### curl（快速测试）

```bash
# 需要安装 websocat 或 wscat
# npm install -g wscat
wscat -c ws://localhost:9081/ws
```

---

## 服务端 API

### 全局 Hub

```go
import "modbus/websocket"

// 获取全局 Hub 实例
hub := websocket.GlobalHub
```

### `hub.Broadcast(msg []byte)`

向所有在线客户端广播消息：

```go
hub.Broadcast([]byte("系统公告：服务即将重启"))
```

### `hub.Count() int`

获取当前在线客户端数量：

```go
count := hub.Count()
log.Printf("当前在线: %d", count)
```

### `hub.HandleWebSocket(w, r)`

WebSocket 握手处理器。已在 `run.go` 中自动注册到 `/ws`，一般无需手动调用。

---

## 接收客户端消息（自定义处理）

编辑 `client.go` 中的 `ReadPump` 方法，在 `// Todo` 处添加业务逻辑：

```go
// client.go — ReadPump()
log.Printf("📩 [WebSocket] 收到来自 %s 的消息: %s", c.ID, string(message))

// TODO: 在这里处理收到的消息

// 示例：按消息内容路由
switch string(message) {
case "ping":
    c.Send <- []byte("pong")
case "status":
    c.Send <- []byte(fmt.Sprintf(`{"online":%d}`, c.Hub.Count()))
default:
    // 广播给所有其他客户端
    c.Hub.Broadcast(message)
}
```

---

## 配置项

通过 `data/.env` 文件配置：

| 键 | 默认值 | 说明 |
|---|--------|------|
| `WEBSOCKET_PATH` | `/ws` | WebSocket 连接路径 |

修改路径（如改为 `/socket`）：

```bash
# data/.env
WEBSOCKET_PATH=/socket
```

重启后客户端连接地址变为 `ws://localhost:9081/socket`。

---

## 客户端 API

### `Client` 结构体

| 字段 | 类型 | 说明 |
|------|------|------|
| `ID` | `string` | 客户端唯一标识（UUID） |
| `Conn` | `*websocket.Conn` | WebSocket 连接对象 |
| `Hub` | `*Hub` | 所属 Hub |
| `Send` | `chan []byte` | 发送消息缓冲通道 |

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
