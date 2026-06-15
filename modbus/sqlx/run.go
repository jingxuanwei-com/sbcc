package sqlx

import (
	"fmt"
	"modbus/env"
	"modbus/sql"
	"os"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/glebarez/go-sqlite"  // 纯 Go SQLite
	_ "github.com/go-sql-driver/mysql" // 纯 Go MySQL
	_ "github.com/jackc/pgx/v5/stdlib" // 纯 Go PgSQL
)

var DB *sqlx.DB

func Run() {
	var driver string
	switch env.Get("DB_TYPE") {
	case "sqlite", "sqllite", "sqlite3":
		driver = "sqlite"
	case "pgsql", "postgres", "postgresql":
		driver = "pgx"
	case "mysql", "mariadb":
		driver = "mysql"
	default:
		fmt.Printf("⚠️ [SQLX] 数据库模块 配置错误！无法识别的数据库类型:%s\n", env.Get("DB_TYPE"))
		// 程序退出
		os.Exit(1)
		return
	}
	// 直接使用已经连接成功的 Sql 实例进行包装
	// 第二个参数是驱动名称（如 "sqlite" 或 "mysql"），sqlx 需要它来决定 SQL 占位符风格
	DB = sqlx.NewDb(sql.DB, driver)

	// 间隔 2 秒
	time.Sleep(2 * time.Second)

	// // 开始无限循环重连检测
	for {
		// 测试连接
		err := DB.Ping()
		if err == nil {
			fmt.Printf("✅ [SQLX] 数据库模块 连接成功！\n")
			break // 连上了，跳出循环
		}
		fmt.Printf("🔄 [SQLX] 数据库连接失败: %v。 9秒后尝试重连...\n", err)
		time.Sleep(9 * time.Second)

	}

}
