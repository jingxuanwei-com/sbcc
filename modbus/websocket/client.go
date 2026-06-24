// ============================================================
//
//	WebSocket 客户端封装 — SBCC 控制中心
//	功能：管理单个 WebSocket 连接的生命周期（读写、心跳、关闭）
//	依赖：gorilla/websocket
//	启动方式：由 Hub.HandleWebSocket 自动创建
//
//	📖 一个客户端 = 一条"电话线"
//	每个浏览器连上后，都会创建一个人 Client 对象。
//	然后派两个"工人"全程守着这条线：
//	  - WritePump: 专门发消息给这个浏览器
//	  - ReadPump:  专门收这个浏览器发来的消息
//
// ============================================================
package websocket

import (
	"log"  // 日志输出
	"time" // 时间控制（心跳、超时）

	"github.com/google/uuid"       // UUID：生成唯一的客户端 ID
	"github.com/gorilla/websocket" // WebSocket 库
)

// ═══════════════════════════════════════════════════════════════
//
//	常量配置
//	这些是 WebSocket 连接的各种"规矩"——超时多久、包多大等
//
// ═══════════════════════════════════════════════════════════════
const (
	// writeWait: 发消息时最多等多久
	// 如果发一条消息花了超过 10 秒，就认为连接断了
	writeWait = 10 * time.Second

	// pongWait: 多久没收到客户端的"我还活着"信号就算断
	// 客户端收到 ping 后应该回一个 pong
	// 如果 60 秒没收到 pong，就认为客户端挂了
	pongWait = 60 * time.Second

	// pingPeriod: 每隔多久给客户端发一次"你还活着吗？"
	// 必须小于 pongWait，一般设成 pongWait 的 90%
	// 60秒 * 0.9 = 54秒发一次 ping
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize: 客户端发来的消息最大允许多大
	// 超过 4KB 的消息就不收了（防止有人搞破坏）
	maxMessageSize = 4096

	// sendBufferSize: 发送缓冲区能缓存多少条消息
	// 如果发消息比客户端收消息快，消息先在这里排队
	// 排满 256 条还发不出去，就丢掉（总比堵死好）
	sendBufferSize = 256
)

// ═══════════════════════════════════════════════════════════════
//
//	Client — 一个 WebSocket 客户端连接
//
//	每个连上来的浏览器都对应一个 Client 对象。
//	它包含了：
//	- ID:      这个客户端的身份证号
//	- Conn:    真正的 WebSocket 电话线
//	- Hub:     它属于哪个聊天室
//	- Send:    它的私人信箱（通道），Broadcast 往这里投信
//
// ═══════════════════════════════════════════════════════════════
type Client struct {
	// ID: 客户端唯一标识（UUID 格式）
	// 用来区分不同用户，比如 "a1b2c3d4-e5f6-..."
	ID string

	// Conn: WebSocket 连接对象（真正的电话线）
	// 读它 → 收消息，写它 → 发消息
	Conn *websocket.Conn

	// Hub: 这个客户端属于哪个聊天室管理员
	// 客户端断开时需要通过 Hub 从花名册里注销
	Hub *Hub

	// Send: 消息通道（私人信箱）
	// 容量 = sendBufferSize = 256 条
	//
	// 工作方式：
	//   Hub.Broadcast → 往每个客户端的 Send 投一条消息
	//   WritePump     → 从 Send 取出消息，写到 WebSocket
	//
	// 这就像"邮递员往你家信箱投信，你从信箱拿信看"
	Send chan []byte
}

// ═══════════════════════════════════════════════════════════════
//
//	NewClient — 创建一个新的客户端
//
//	当用户连上 WebSocket 时，会调用这个函数。
//	它会：
//	1. 生成一个唯一的 ID（UUID）
//	2. 存好电话线（Conn）
//	3. 记下属于哪个 Hub
//	4. 初始化一个信箱（Send 通道）
//
// ═══════════════════════════════════════════════════════════════
func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:   uuid.New().String(),               // 生成唯一 ID
		Conn: conn,                              // 存好电话线
		Hub:  hub,                               // 记下属于哪个聊天室
		Send: make(chan []byte, sendBufferSize), // 准备一个信箱
	}
}

// ═══════════════════════════════════════════════════════════════
//
//	ReadPump — 「读取泵」专门收消息的工人
//
//	这个工人全程守着电话线，等客户端发消息过来。
//	收到消息后：
//	1. 如果 Hub 设置了 OnMessage 回调 → 交给回调处理
//	2. 如果没设置 → 只打印日志
//
//	如果客户端断线了，工人会自动把客户端从花名册里删除。
//
//	注意：这是一个死循环，一直在跑，直到连接断开。
//
// ═══════════════════════════════════════════════════════════════
func (c *Client) ReadPump() {
	// defer: 不管函数怎么结束，最后都会执行这里
	// 客户端断开时，告诉 Hub "这个人走了"
	defer func() {
		c.Hub.Unregister(c.ID)
	}()

	// ─── 设置读取规矩 ────────────────────────────────────
	c.Conn.SetReadLimit(maxMessageSize)              // 消息最大 4KB
	c.Conn.SetReadDeadline(time.Now().Add(pongWait)) // 60秒没消息就超时
	c.Conn.SetPongHandler(func(string) error {       // 收到 pong 的处理
		c.Conn.SetReadDeadline(time.Now().Add(pongWait)) // 收到 pong 重置超时
		return nil
	})

	// ─── 循环读取 ─────────────────────────────────────────
	for {
		// ReadMessage: 阻塞等待，直到收到消息或连接断开
		// 返回的消息类型（TextMessage/BinaryMessage）、内容、错误
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			// 出错了，看看是不是正常断开
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,         // 正在离开
				websocket.CloseNormalClosure,     // 正常关闭
				websocket.CloseAbnormalClosure) { // 异常关闭
				log.Printf("⚠️ [WebSocket] 客户端 %s 异常断开: %v", c.ID, err)
			}
			break // 不管正不正常，都退出循环
		}

		// ─── 处理收到的消息 ──────────────────────────
		if c.Hub.OnMessage != nil {
			// 如果模块设置了"收到消息干啥"，就调它
			c.Hub.OnMessage(c, message)
		} else {
			// 没设置的话，就只是打印出来看看
			log.Printf("📩 %s 收到来自 %s 的消息: %s", c.Hub.logTag(), c.ID, string(message))
		}
	}
}

// ═══════════════════════════════════════════════════════════════
//
//	WritePump — 「写入泵」专门发消息的工人
//
//	这个工人一直盯着信箱（Send 通道）。
//	只要有人往信箱里投信（Broadcast 发消息），
//	它就取出来通过 WebSocket 发给客户端。
//
//	除了发消息，它还定期发"心跳包"（Ping）确认客户端还活着。
//
//	注意：这也是一个死循环，一直在跑，直到连接断开。
//
// ═══════════════════════════════════════════════════════════════
func (c *Client) WritePump() {
	// ticker: 定时器，每 54 秒响一次，提醒该发心跳了
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()  // 关闭定时器
		c.Conn.Close() // 关闭 WebSocket 连接
	}()

	// ─── 循环等待 ─────────────────────────────────────────
	// select 同时等两件事：有消息要发 / 该发心跳了
	for {
		select {
		case message, ok := <-c.Send:
			// ── 信箱里有信！取出来发给客户端 ─────
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait)) // 发消息限时 10 秒

			if !ok {
				// ok == false 表示通道被关闭了（客户端已断开）
				// 发一个"关闭"帧告诉浏览器：这边要挂了
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// NextWriter: 获取一个写入器
			// 就像拿一支笔，往 WebSocket 里写文字
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return // 拿不到笔，说明连接有问题了
			}
			w.Write(message) // 先写当前这条消息

			// ── 看看信箱里还有没有别的信 ─────────
			// 如果有积压，一口气全发了（减少网络开销）
			n := len(c.Send) // 看看排队还有几条
			for i := 0; i < n; i++ {
				w.Write(<-c.Send) // 取出来，写进去
			}

			if err := w.Close(); err != nil {
				return // 写完收笔，但要检查有没有写失败
			}

		case <-ticker.C:
			// ── 到点了，发个心跳 Ping ────────────
			// 告诉客户端："我还活着，你呢？"
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return // 发心跳失败，说明连接断了
			}
		}
	}
}
