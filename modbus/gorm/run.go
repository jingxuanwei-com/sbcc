package gorm

import (
	"fmt"
	"modbus/env"
	"modbus/sql"
	"os"

	_ "github.com/glebarez/go-sqlite" // 纯 Go SQLite
	"github.com/glebarez/sqlite"
	_ "github.com/go-sql-driver/mysql" // 纯 Go MySQL
	_ "github.com/jackc/pgx/v5/stdlib" // 纯 Go PgSQL
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Run 接收一个已经连接好的原生 *sql.DB 句柄和驱动名称
func Run() {
	var dialector gorm.Dialector

	switch env.Get("DB_TYPE") {
	case "sqlite", "sqllite", "sqlite3":
		dialector = sqlite.Dialector{Conn: sql.DB}

	case "pgsql", "postgres", "postgresql":
		dialector = postgres.New(postgres.Config{Conn: sql.DB})

	case "mysql", "mariadb":
		dialector = mysql.New(mysql.Config{Conn: sql.DB})

	default:
		fmt.Printf("⚠️ [GROM] 数据库模块 配置错误！无法识别的数据库类型:%s\n", env.Get("DB_TYPE"))
		// 程序退出
		os.Exit(1)
		return
	}

	// 初始化 GORM 实例
	var err error
	DB, err = gorm.Open(dialector, &gorm.Config{
		// 这里可以根据需要配置 GORM 的行为
		SkipDefaultTransaction: true, // 如果追求性能，可以跳过默认事务
	})

	if err != nil {
		fmt.Printf("❌ [GORM] 绑定原生连接失败: %v\n", err)
	}

	fmt.Printf("✅ [GORM] 已成功挂载到原生 %s 连接池\n", env.Get("DB_TYPE"))
}
