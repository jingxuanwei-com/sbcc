# Server 模块

基于 `net/http` 的 web 服务启动模块，路由引擎来自 web 模块。只负责启动，不提供重启 / 重载 / 状态查询。

## 配置

读取 `data/config.yaml` 的 `server` 段：

```yaml
server:
  ip: ""                  # 空表示监听所有网卡
  port: 9081
  https: false
  cert_file: ""           # https=true 时必填
  key_file: ""            # https=true 时必填
```

- `https: false`：启动 HTTP
- `https: true` 且 `cert_file`、`key_file` 都有值：启动 HTTPS
- `https: true` 但证书缺失：打印警告并回退 HTTP

## 使用方法

```go
config.Start() // 读取 data/config.yaml
server.Start() // 阻塞运行，直到进程退出
```

## 行为说明

- `ReadTimeout` 5s，`WriteTimeout` 10s
- 监听失败直接 `log.Fatalf` 退出，不静默重试
- 修改配置后需重启进程生效，不做热重载
