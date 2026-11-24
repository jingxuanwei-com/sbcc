// Package db 数据库模块：持有全局 DB 实例，负责数据库连接的建立/关闭/重连（ORM 底层为 gorm）。
// 表结构定义与迁移在 model 模块。
package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/1szt/go-pkg/env"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB
var lastConfig string // 上次连接时的配置指纹

type DBStatus struct {
	Type   string `json:"type"`
	Host   string `json:"host"`
	Port   string `json:"port"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func Start() {
	if env.Get("SBCC_SERVER_INSTALL") != "true" {
		fmt.Println("⚠️ [DB] 服务器未初始化，跳过数据库连接，等待初始化...")
		return
	}
	connect()
}

func Stop() {
	if DB == nil {
		return
	}
	sqlDB, err := DB.DB()
	if err != nil {
		fmt.Printf("❌ [DB] 获取底层连接失败: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		fmt.Printf("❌ [DB] 关闭数据库连接失败: %v", err)
		return
	}
	DB = nil
	fmt.Printf("✅ [DB] 数据库连接已关闭")
}

func Restart() {
	Stop()
	connect()
}

func Reload() {
	// 构造当前配置指纹
	currentConfig := configFingerprint()

	// 检查数据库是否已连接
	connected := false
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			if err = sqlDB.Ping(); err == nil {
				connected = true
			}
		}
	}

	// 配置没变且已连接 → 跳过
	if connected && currentConfig == lastConfig {
		fmt.Printf("🔄 [DB] 配置无变化且已连接，跳过重载")
		return
	}

	// 有变化或未连接 → 重连
	if !connected {
		fmt.Printf("🔄 [DB] 数据库未连接，重新连接...")
	} else {
		fmt.Printf("🔄 [DB] 配置已变化，重新连接...")
	}
	Stop()
	connect()
}

// configFingerprint 根据数据库相关配置生成指纹，用于判断配置是否变化
func configFingerprint() string {
	dbType := env.Get("SBCC_DB_TYPE")
	return dbType + "|" + env.Get("SBCC_DB_HOST") + "|" + env.Get("SBCC_DB_PORT") + "|" + env.Get("SBCC_DB_NAME") + "|" + env.Get("SBCC_DB_PATH")
}

func Status() DBStatus {
	dbType := env.Get("SBCC_DB_TYPE")
	status := "disconnected"
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			if err = sqlDB.Ping(); err == nil {
				status = "connected"
			}
		}
	}
	return DBStatus{
		Type:   dbType,
		Host:   env.Get("SBCC_DB_HOST"),
		Port:   env.Get("SBCC_DB_PORT"),
		Name:   env.Get("SBCC_DB_NAME"),
		Status: status,
	}
}

func connect() {
	dbType := env.Get("SBCC_DB_TYPE")
	if dbType == "" {
		fmt.Printf("❌ [DB] 未找到 SBCC_DB_TYPE 配置")
		return
	}

	var dialector gorm.Dialector

	switch dbType {
	case "sqlite", "sqlite3":
		path := env.Get("SBCC_DB_PATH")
		// 自动创建数据库文件所在目录，避免 "unable to open database file"
		if dir := filepath.Dir(path); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("❌ [DB] 创建数据库目录 %s 失败: %v", dir, err)
				return
			}
		}
		dialector = sqlite.Open(path)

	case "pgsql", "postgres", "postgresql":
		host := env.Get("SBCC_DB_HOST")
		port := env.Get("SBCC_DB_PORT")
		user := env.Get("SBCC_DB_USER")
		password := env.Get("SBCC_DB_PASSWORD")
		name := env.Get("SBCC_DB_NAME")
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, name)
		dialector = postgres.Open(dsn)

	case "mysql", "mariadb":
		host := env.Get("SBCC_DB_HOST")
		port := env.Get("SBCC_DB_PORT")
		user := env.Get("SBCC_DB_USER")
		password := env.Get("SBCC_DB_PASSWORD")
		name := env.Get("SBCC_DB_NAME")
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, name)
		dialector = mysql.Open(dsn)

	default:
		fmt.Printf("⚠️ [DB] 数据库模块 配置错误！无法识别的数据库类型: %s", dbType)
		return
	}

	var err error
	DB, err = gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})

	if err != nil {
		fmt.Printf("❌ [DB] 初始化数据库 %s 失败: %v", dbType, err)
		return
	}

	fmt.Printf("✅ [DB] 已成功初始化数据库 %s", dbType)
	lastConfig = configFingerprint()
}
