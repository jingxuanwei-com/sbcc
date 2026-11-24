# web 模块

持有全局路由引擎 `Engine`、注册全局中间件。必须在所有业务模块注册路由之前调用 `web.Start()`（gin 中间件只对之后注册的路由生效）。

## 目录结构

```text
web/
├── web.go                    # Engine + Start()：唯一装配点，中间件按序 Use(...)
└── middleware/               # 包 middleware：全局中间件集合
    └── init_guard.go         # InitGuard 初始化检查（一个中间件一个文件）
```

## 中间件演进规则（为上百个中间件做准备）

1. **默认平铺**：新中间件 = `middleware/` 下加一个文件，导出 `func Xxx() gin.HandlerFunc`
2. **按域拆子包**：某个域膨胀到 2~3 个文件（如 auth 有 JWT/RBAC/刷新令牌）才建子目录，目录名即域前缀：`middleware/auth/`、`middleware/ratelimit/`
3. **`web.Start()` 是唯一装配点**：所有中间件在 `Start()` 里按序 `Use()`，启动顺序一眼可见
4. **依赖单向**：`middleware → config/db` 允许（读业务配置）；`middleware → web` 禁止（要 Engine 说明放错位置，装配在 web 层做）

## InitGuard：全局初始化检查中间件

未完成初始化（`server_install != "true"`）时拦截请求，返回统一的未初始化提示，业务子包不需要各自重复判断。

### 用法

`web.Start()` 里已自动挂载，无需手动调用：

```go
func Start() {
	Engine.Use(gin.Logger(), gin.Recovery(), middleware.InitGuard())
}
```

默认放行（未初始化也能访问）：

- `/`（根路径，安装页入口）
- `/api/install/` 前缀（安装接口自身，check / submit）

要增减白名单，改 [init_guard.go](./middleware/init_guard.go) 顶部的 `initFreePaths` / `initFreePrefixes`。

### 被拦截时的返回结构

HTTP 状态码 `503 Service Unavailable`：

```json
{
  "code": 40001,
  "message": "服务器未初始化，请先完成初始化",
  "need_install": true
}
```

| 字段 | 说明 |
| --- | --- |
| `code` | 固定 `40001`，前端据此跳转安装页 |
| `message` | 可直接展示给用户的提示文案 |
| `need_install` | 恒为 `true`，与 `/api/install/check` 返回字段对齐 |

已初始化时中间件直接放行，不影响任何请求。
