// Package db 数据库模块：持有全局 DB 实例，负责数据库连接的建立与关闭（ORM 底层为 gorm）。
// 表结构定义与迁移在 model 模块。
package db

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"server/core/config"
)

// DB 全局数据库实例
var DB *gorm.DB

// Start 按 data/config.yaml 的 db 配置连接数据库；未完成初始化时跳过。
func Start() {
	if !config.Cfg.Install {
		fmt.Println("⚠️ [DB] 服务器未初始化，跳过数据库连接，等待初始化...")
		return
	}

	d := config.Cfg.DB

	var dialector gorm.Dialector
	switch d.Type {
	case "sqlite", "sqlite3":
		// 自动创建数据库文件所在目录，避免 "unable to open database file"
		if dir := filepath.Dir(d.Path); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("❌ [DB] 创建数据库目录 %s 失败: %v", dir, err)
				return
			}
		}
		dialector = sqlite.Open(d.Path)

	case "pgsql", "postgres", "postgresql":
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", d.Host, d.Port, d.User, d.Password, d.Name)
		dialector = postgres.Open(dsn)

	case "mysql", "mariadb":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", d.User, d.Password, d.Host, d.Port, d.Name)
		dialector = mysql.Open(dsn)

	default:
		fmt.Printf("⚠️ [DB] 无法识别的数据库类型: %s", d.Type)
		return
	}

	var err error
	DB, err = gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		fmt.Printf("❌ [DB] 初始化数据库 %s 失败: %v", d.Type, err)
		return
	}

	fmt.Printf("✅ [DB] 已成功初始化数据库 %s", d.Type)
}

// Stop 关闭数据库连接
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
