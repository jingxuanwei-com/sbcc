# SCS Session 管理模块

基于 `github.com/alexedwards/scs/v2`，使用数据库（SQLite / PostgreSQL / MySQL）做 Session 存储后端。

## 目录

- [模块职责](#模块职责)
- [启动流程](#启动流程)
- [配置项](#配置项)
- [核心函数](#核心函数)
- [中间件](#中间件)
- [完整示例](#完整示例)
  - [登录](#登录)
  - [注销](#注销)
  - [查看登录状态](#查看登录状态)
  - [注册并自动登录](#注册并自动登录)
  - [受保护路由](#受保护路由)
- [Session 生命周期](#session-生命周期)
- [数据库表结构](#数据库表结构)
- [注意事项](#注意事项)

---

## 模块职责

仅做一件事：**提供 HTTP Session 管理**。

- 登录后把 `user_id`、`username` 等写入 Session
- 后续请求通过 Cookie 自动读取 Session 数据
- 注销时销毁 Session
- 使用 `modbus/sql` 的公共 `sql.DB` 作为存储后端

---

## 启动流程

```go
import "modbus/scs"

// 在 main() 中按顺序：
sql.Run()    // 先初始化数据库
scs.Run()    // 再初始化 Session
user.Run()   // 最后启动用户模块
```

`scs.Run()` 会做三件事：

1. 读取环境变量配置过期时间
2. 创建 `sessions` 表（如不存在）
3. 根据 `DB_TYPE` 选择对应的 SCS 存储后端

---

## 配置项

通过 `modbus/env` 读取环境变量，支持 `.env` 文件或系统环境变量：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `SESSION_LIFETIME` | `86400`（24小时） | Session 绝对过期时间（秒） |
| `SESSION_IDLE_TIMEOUT` | `0` | 空闲超时（秒），`0`=不限制 |

---

## 核心函数

| 函数 | 说明 |
|---|---|
| `scs.Scs.LoadAndSave` | HTTP 中间件，自动从 Cookie 加载/保存 Session |
| `scs.Scs.Put(ctx, key, val)` | 向当前 Session 写入值 |
| `scs.Scs.Get(ctx, key)` | 从当前 Session 读取值（返回 `any`） |
| `scs.Scs.GetString(ctx, key)` | 读取 string |
| `scs.Scs.GetInt64(ctx, key)` | 读取 int64 |
| `scs.Scs.GetFloat(ctx, key)` | 读取 float64 |
| `scs.Scs.GetBool(ctx, key)` | 读取 bool |
| `scs.Scs.Pop(ctx, key)` | 读取后删除 |
| `scs.Scs.Exists(ctx, key)` | 检查 key 是否存在 |
| `scs.Scs.Keys(ctx)` | 返回当前 Session 的所有 key |
| `scs.Scs.RenewToken(ctx)` | 更换 Session token（防 Session 固定攻击） |
| `scs.Scs.Destroy(ctx)` | 销毁当前 Session |
| `scs.Scs.DestroyAll(ctx)` | 销毁当前用户的所有 Session |

完整 API 参考：[SCS 官方文档](https://github.com/alexedwards/scs)

---

## 中间件

所有需要读写 Session 的路由组，必须先挂载 `scs.Scs.LoadAndSave`：

```go
r.Group(func(r chi.Router) {
    r.Use(scs.Scs.LoadAndSave)  // ← 必须

    r.Post("/login", handleLogin)
    r.Post("/logout", handleLogout)
    r.Get("/me", handleMe)
})
```

---

## 完整示例

### 登录

```go
func handleLogin(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // 校验用户名密码（从数据库查询）...

    // 1. 续命 token（防 Session 固定攻击）
    scs.Scs.RenewToken(r.Context())

    // 2. 写入 Session
    scs.Scs.Put(r.Context(), "user_id", user.ID)
    scs.Scs.Put(r.Context(), "username", user.Username)

    // 3. SCS 自动设置 Cookie，前端无需额外操作
    writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
```

**前端（浏览器）**：浏览器自动携带 Cookie，无需处理。

**前端（fetch）**：默认同域请求会自动带 Cookie。

---

### 注销

```go
func handleLogout(w http.ResponseWriter, r *http.Request) {
    if err := scs.Scs.Destroy(r.Context()); err != nil {
        http.Error(w, "注销失败", http.StatusInternalServerError)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
```

---

### 查看登录状态

```go
func handleMe(w http.ResponseWriter, r *http.Request) {
    userID := scs.Scs.GetInt64(r.Context(), "user_id")
    username := scs.Scs.GetString(r.Context(), "username")

    if userID == 0 {
        writeJSON(w, http.StatusOK, map[string]any{"logged_in": false})
        return
    }

    writeJSON(w, http.StatusOK, map[string]any{
        "logged_in": true,
        "user_id":   userID,
        "username":  username,
    })
}
```

---

### 注册并自动登录

```go
func handleRegister(w http.ResponseWriter, r *http.Request) {
    // 创建用户写入数据库...

    // 注册成功后自动登录（续命 + 写入）
    scs.Scs.RenewToken(r.Context())
    scs.Scs.Put(r.Context(), "user_id", user.ID)
    scs.Scs.Put(r.Context(), "username", user.Username)

    writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}
```

---

### 受保护路由

```go
// 中间件
func RequireLogin(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := scs.Scs.GetInt64(r.Context(), "user_id")
        if userID == 0 {
            writeJSON(w, http.StatusUnauthorized, map[string]any{
                "ok": false, "message": "请先登录",
            })
            return
        }
        next.ServeHTTP(w, r)
    })
}

// 使用
r.Group(func(r chi.Router) {
    r.Use(scs.Scs.LoadAndSave)
    r.Use(RequireLogin)

    r.Get("/profile", handleProfile)
})
```

---

## Session 生命周期

```mermaid
sequenceDiagram
    participant 客户端
    participant 服务端
    participant 数据库

    客户端->>服务端: POST /api/user (username + password)
    服务端->>数据库: 查询用户
    服务端->>数据库: 写入 Session (token, data, expiry)
    服务端-->>客户端: Set-Cookie: session=<token>
    客户端->>服务端: GET /api/user/me (Cookie: session=<token>)
    服务端->>数据库: 查询 Session
    服务端-->>客户端: { logged_in: true, username: "..." }
    客户端->>服务端: POST /api/user/logout
    服务端->>数据库: 删除 Session
    服务端-->>客户端: Set-Cookie: session=; Max-Age=0
```

- **创建**：登录时 `RenewToken` + `Put`
- **读取**：每次请求 `LoadAndSave` 中间件自动从 Cookie 读取 token，查数据库加载数据
- **续期**：`RenewToken` 更换 token 值，同时刷新过期时间
- **销毁**：`Destroy` 从数据库删除，并清除客户端 Cookie

---

## 数据库表结构

根据 `DB_TYPE` 自动创建 `sessions` 表：

**SQLite**
```sql
CREATE TABLE IF NOT EXISTS sessions (
    token  TEXT PRIMARY KEY,
    data   BLOB    NOT NULL,
    expiry REAL    NOT NULL
);
```

**PostgreSQL**
```sql
CREATE TABLE IF NOT EXISTS sessions (
    token  TEXT PRIMARY KEY,
    data   BYTEA         NOT NULL,
    expiry TIMESTAMPTZ   NOT NULL
);
```

**MySQL / MariaDB**
```sql
CREATE TABLE IF NOT EXISTS sessions (
    token  VARCHAR(255) PRIMARY KEY,
    data   BLOB         NOT NULL,
    expiry TIMESTAMP(6) NOT NULL
);
```

> SCS 的清理机制：`sessions` 表的数据由 SCS 内部自动清理（基于 `expiry` 字段），无需手动维护。

---

## 注意事项

1. **启动顺序**：`scs.Run()` 必须在 `sql.Run()` 之后，否则数据库未准备好
2. **中间件**：所有读写 Session 的路由组必须挂载 `scs.Scs.LoadAndSave`
3. **Session 固定攻击**：登录成功后务必调用 `RenewToken`（更换 token 值）
4. **跨域**：如果前端不同域，需配置 `Scs.Cookie.Domain` 和 `Scs.Cookie.Secure` 等参数
5. **仅支持 Cookie**：SCS 是纯 Cookie 方案，移动端/小程序需通过浏览器 WebView 或改用 token 方案
6. **并发安全**：SCS 内部通过数据库行锁保证并发安全
