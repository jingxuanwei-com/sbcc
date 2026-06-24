// ============================================================
//
//	WebSocket Hub 连接管理器 — SBCC 控制中心
//	功能：管理所有 WebSocket 客户端连接，支持广播、注册、注销
//	依赖：gorilla/websocket
//	使用方式：由 websocket.Run() 自动创建，各模块通过 GlobalHub 使用
//
//	📖 什么是 Hub？
//	Hub 就像一个"聊天室管理员"——它管着所有连上来的客户端，
//	谁来了（注册）、谁走了（注销）、要给所有人发消息（广播），
//	都是 Hub 的活。
//
// ============================================================
package websocket

import (
	"encoding/json" // JSON 编解码：把数据转成 JSON 字符串
	"log"           // 日志输出：打印到控制台
	"net/http"      // HTTP 处理：处理浏览器的请求
	"sync"          // 并发安全锁：防止多个协程同时读写出问题

	"github.com/gorilla/websocket" // WebSocket 库：处理 WebSocket 协议
)

// ═══════════════════════════════════════════════════════════════
//
//	Hub — WebSocket 客户端连接管理器
//
//	Hub 就是一个"容器"，里面存着所有当前在线的客户端。
//	你可以往里面加人（Register）、减人（Unregister）、
//	给所有人喊话（Broadcast）。
//
//	现在项目用的是全局统一的 GlobalHub，所有模块共用一个。
//	但如果需要，你也可以创建独立的 Hub（比如不同模块想隔离）。
//
// ═══════════════════════════════════════════════════════════════
type Hub struct {
	// ─── 公开字段（别的包能直接读写） ─────────────────────

	// Name: Hub 的名字，用来区分日志是谁打的
	// 比如 GlobalHub 打日志就显示 [WebSocket-Global]
	Name string

	// OnMessage: 收到客户端消息时的"回调函数"
	// 如果设置了，客户端发消息来就会调这个函数
	// 没设置就只打印日志，不处理
	OnMessage func(client *Client, data []byte)

	// ─── 私有字段（只有 Hub 自己能读写） ─────────────────

	// mu: 读写锁，防止多个协程同时操作 clients 出问题
	// Go 的 map 在并发读写时会崩溃，这个锁就是保护 map 的
	mu sync.RWMutex

	// clients: 存着所有在线客户端
	// key = 客户端ID, value = 客户端对象
	// 就像一张花名册，记着所有人的名字和电话
	clients map[string]*Client

	// upgrader: HTTP 升级器
	// 它的作用是把普通的 HTTP 请求"升级"成 WebSocket 长连接
	// 就像把一封信（HTTP）变成一条电话线（WebSocket）
	upgrader websocket.Upgrader
}

// ═══════════════════════════════════════════════════════════════
//  统一消息协议
//
//  所有通过 WebSocket 发的消息都用这个格式：
//  {
//    "type":    "模块名.事件名",   ← 前端用这个区分消息来源
//    "payload": { ... }           ← 实际数据
//  }
//
//  例子：
//  {"type":"home.random", "payload":{"time":"14:30:00","value":42}}
//  {"type":"podman.status","payload":{"containers":3,"running":2}}
// ═══════════════════════════════════════════════════════════════

// Message 标准 WebSocket 消息结构
// Type:    "模块名.事件名"，点号分隔，如 "home.random"、"podman.status"
// Payload: 真正的数据，每个模块自己定义
type Message struct {
	Type    string      `json:"type"`    // 消息类型，前端用 switch 判断
	Payload interface{} `json:"payload"` // 实际数据，可以是任何东西
}

// ═══════════════════════════════════════════════════════════════
//
//	GlobalHub — 全局统一的 WebSocket Hub
//
//	这是整个项目的"总聊天室"：
//	- Home 模块往里发 "home.random"
//	- Podman 模块往里发 "podman.status"
//	- 前端只需要连一条 WebSocket，就能收所有模块的消息
//
//	不用 new，直接 websocket.GlobalHub 就行
//
// ═══════════════════════════════════════════════════════════════
var GlobalHub = NewHub("Global")

// ═══════════════════════════════════════════════════════════════
//
//	BroadcastToType — 按类型广播消息（推荐使用）
//
//	这是模块发消息的"标准姿势"：
//	1. 你给一个 type 和 payload
//	2. 它帮你打包成 {"type":"...","payload":{...}} 格式
//	3. 再通过 Broadcast 发给所有客户端
//
//	用法：
//	  GlobalHub.BroadcastToType("home.random", map[string]interface{}{...})
//
//	和 Broadcast 的区别：
//	  BroadcastToType: 自动帮你打包 JSON，加上 type 字段
//	  Broadcast:       你自己打包好原始字节，它只管发
//
// ═══════════════════════════════════════════════════════════════
func (h *Hub) BroadcastToType(msgType string, payload interface{}) {
	// json.Marshal 把数据变成 JSON 字符串（字节数组）
	msg, err := json.Marshal(Message{Type: msgType, Payload: payload})
	if err != nil {
		log.Printf("❌ %s 序列化消息失败: %v", h.logTag(), err)
		return // 转 JSON 失败就放弃，别崩
	}
	// 调用底层的 Broadcast 发出去
	h.Broadcast(msg)
}

// ═══════════════════════════════════════════════════════════════
//
//	NewHub — 创建一个新的 Hub
//
//	用法：hub := websocket.NewHub("我的模块")
//	参数 name 用在日志里，方便看是哪个 Hub 在干活
//
//	一般情况下用 GlobalHub 就够了，不需要自己 new
//
// ═══════════════════════════════════════════════════════════════
func NewHub(name string) *Hub {
	return &Hub{
		Name:    name,
		clients: make(map[string]*Client), // 初始化空的花名册
		upgrader: websocket.Upgrader{
			// CheckOrigin: 检查是从哪个网站连过来的
			// 返回 true = 允许所有来源（开发时方便）
			// 生产环境应该改成只允许你自己的域名
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024, // 读缓冲区 1KB
			WriteBufferSize: 1024, // 写缓冲区 1KB
		},
	}
}

// ═══════════════════════════════════════════════════════════════
//
//	logTag — 生成日志标签
//
//	比如 Hub 叫 "Global"，就返回 "[WebSocket-Global]"
//	这样打日志时能一眼看出是哪个 Hub 在说话
//
// ═══════════════════════════════════════════════════════════════
func (h *Hub) logTag() string {
	if h.Name == "" {
		return "[WebSocket]"
	}
	return "[WebSocket-" + h.Name + "]"
}

// ═══════════════════════════════════════════════════════════════
//
//	Register — 注册客户端（有人连接时调用）
//
//	当浏览器连上 WebSocket 时，把这个人记到花名册里。
//	之后 Broadcast 就知道要发给谁了。
//
//	流程：
//	  HandleWebSocket → NewClient → Register → 上线！
//
// ═══════════════════════════════════════════════════════════════
func (h *Hub) Register(client *Client) {
	// Lock: 加锁，防止多个协程同时写 clients 导致崩溃
	h.mu.Lock()
	defer h.mu.Unlock() // Unlock: 函数结束时自动解锁

	// 把客户端存到 map 里，key 是客户端 ID
	h.clients[client.ID] = client
	log.Printf("🔗 %s 客户端已连接: %s（当前在线: %d）", h.logTag(), client.ID, len(h.clients))
}

// ═══════════════════════════════════════════════════════════════
//
//	Unregister — 注销客户端（有人断开时调用）
//
//	浏览器关闭页面、断网、刷新时，把这个人从花名册里删掉。
//	之后 Broadcast 就不会再给他发消息了。
//
// ═══════════════════════════════════════════════════════════════
func (h *Hub) Unregister(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 先看看花名册里有没有这个人
	if client, ok := h.clients[clientID]; ok {
		client.Conn.Close()         // 关闭 WebSocket 连接
		delete(h.clients, clientID) // 从 map 里删除
		log.Printf("🔌 %s 客户端已断开: %s（当前在线: %d）", h.logTag(), clientID, len(h.clients))
	}
	// 注意：如果 clientID 不在花名册里，什么也不做
}

// ═══════════════════════════════════════════════════════════════
//
//	Broadcast — 广播消息（给所有在线客户端发消息）
//
//	给花名册里的每一个人都发一条消息。
//	如果谁的收件箱（Send 通道）满了，就跳过他不发了。
//	（防止某个客户端卡住导致所有人等）
//
//	工作原理：
//	  1. 遍历所有在线客户端
//	  2. 把消息丢进每个客户端的 Send 通道
//	  3. 客户端的 WritePump 协程会从通道取出并写入 WebSocket
//
// ═══════════════════════════════════════════════════════════════
func (h *Hub) Broadcast(message []byte) {
	// RLock: 读锁，允许多个协程同时读，但写的时候不能读
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 遍历花名册，给每个人发消息
	for id, client := range h.clients {
		select {
		case client.Send <- message:
			// 消息成功丢进通道，WritePump 会处理发送
		default:
			// Send 通道满了（客户端收太慢或断线了）
			// 丢消息比卡死好，直接跳过
			log.Printf("⚠️ %s 客户端 %s 发送缓冲区已满，跳过消息", h.logTag(), id)
		}
	}
}

// ═══════════════════════════════════════════════════════════════
//
//	Count — 当前在线人数
//
//	返回花名册里有几个人。
//	模块发消息前可以先用 Count() 看看有没有人收。
//
// ═══════════════════════════════════════════════════════════════
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ═══════════════════════════════════════════════════════════════
//
//	HandleWebSocket — WebSocket 握手入口
//
//	当用户访问 http://你的地址/websocket 时：
//	1. 浏览器说："我要升级成 WebSocket 连接！"
//	2. HandleWebSocket 说："好，给你升级！"
//	3. 创建 Client 对象，注册到 Hub
//	4. 启动两个后台协程：
//	   - WritePump: 专门负责发消息给这个客户端
//	   - ReadPump:  专门负责收这个客户端发来的消息
//
//	这个函数可以直接当作 chi 路由的 handler 使用。
//
// ═══════════════════════════════════════════════════════════════
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade: 把 HTTP 请求升级成 WebSocket 长连接
	// 升级成功后，conn 就是一条"电话线"了
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ %s 升级失败: %v", h.logTag(), err)
		return // 升级失败就放弃，别崩
	}

	// 创建客户端对象，分配唯一 ID
	client := NewClient(conn, h)

	// 注册到 Hub 的花名册里
	h.Register(client)

	// 启动两个协程，一个发消息一个收消息
	go client.WritePump() // 发消息的工人
	go client.ReadPump()  // 收消息的工人
}
