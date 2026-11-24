# Server 模块

Server 模块负责 web 服务的启动/重启/热重载（路由引擎 web.Engine 与全局中间件在 web 模块）。

# 使用方法

server.Start() 启动 web 服务（从配置读取 server_ip + server_port）如果配置不存在默认监听9081端口
server.Restart() 强制重连（停掉旧服务 → 重新读取配置 → 启动）
server.Reload() 智能重连（配置无变化且运行中则跳过，否则重启）
server.Status() 获取服务状态

# Status 返回结构体

IP - 服务监听地址
Port - 服务端口
Status - 服务状态（running/stopped）
