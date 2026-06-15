package main

import (
	"fmt"
	"modbus/gorm"
	"modbus/home"
	"modbus/podman"
	"modbus/sql"
	"modbus/sqlx"
	"modbus/sub"
	"modbus/chi"
)

func main() {

	// motd
	fmt.Println("========================================")
	fmt.Println("             SBCC 控制中心启动！          ")
	fmt.Println("========================================")

	// Web 引擎启动
	chi.Run()

	// 数据库模块启动
	sql.Run()
	sqlx.Run()
	gorm.Run()

	// 功能模块启动
	home.Run()
	sub.Run()
	podman.Run()

	// gin.Run()

	// 阻塞主进程
	select {}
}
