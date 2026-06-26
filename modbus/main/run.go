package main

import (
	"log"
	"modbus/chi"
	"modbus/gorm"
	"modbus/home"
	"modbus/login"
	"modbus/podman"
	"modbus/scs"
	"modbus/sql"
	"modbus/sqlx"
	"modbus/sub"
	"modbus/websocket"
)

func main() {
	// 配置日志格式：只显示时间 HH:MM:SS，不显示完整日期
	log.SetFlags(log.Ltime | log.Lmsgprefix)

	// motd
	log.Print("========================================")
	log.Print("             SBCC 控制中心启动！          ")
	log.Print("========================================")
	log.Print("正在启动各个模块...")

	// chi 引擎启动
	chi.Run()

	// WebSocket 全局端点注册
	websocket.Run()

	// gRPC 服务启动
	// grpc.Run()

	// 数据库模块启动
	sql.Run()
	sqlx.Run()
	gorm.Run()

	// Session 管理（依赖数据库）
	scs.Run()

	// 登录认证模块（依赖 gorm + scs）
	login.Run()

	// 功能模块启动
	home.Run()
	sub.Run()
	podman.Run()

	// gin.Run()

	// 阻塞主进程
	select {}
}
