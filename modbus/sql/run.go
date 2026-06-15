package sql

import (
	"database/sql"
	"fmt"

	"modbus/env"
	"os"
	"time"

	_ "github.com/glebarez/go-sqlite"  // 纯 Go SQLite
	_ "github.com/go-sql-driver/mysql" // 纯 Go MySQL
	_ "github.com/jackc/pgx/v5/stdlib" // 纯 Go PgSQL
)

var DB *sql.DB

func Run() {
	// 一次性初始化数据库配置
	env.Init([][]string{
		{"DB_TYPE", "sqlite", "--- [数据库驱动配置] ---", "支持类型: sqlite (本地文件), pgsql/mysql/mariadb (远程服务器)", "注意: 若选 sqlite，下方 DB_NAME 则为数据库文件名"},
		{"DB_NAME", "data/sbcc.db", "--- [数据库实例/文件名] ---", "SQLite 示例: data/sbcc.db | PG/MySQL 示例: my_trade_db"},
		{"DB_HOST", "127.0.0.1", "--- [网络配置] ---", "仅针对 pgsql/mysql/mariadb，sqlite 模式下请忽略"},
		{"DB_PORT", "5432", "默认端口: pgsql(5432), mysql/mariadb(3306)"},
		{"DB_USER", "postgres", "数据库用户名"},
		{"DB_PASSWORD", "your_password", "数据库密码"},
	})

	// 匹配数据库类型
	var dsn string
	var driver string
	switch env.Get("DB_TYPE") {
	case "sqlite", "sqllite", "sqlite3":
		// 判断文件是否存在 不存在则创建
		if _, err := os.Stat(env.Get("DB_NAME")); os.IsNotExist(err) {
			// 创建文件
			_, err = os.Create(env.Get("DB_NAME"))
			if err != nil {
				fmt.Printf("⚠️ [SQL] 数据库模块 配置错误！创建数据库文件失败！err: %v\n", err)
				// 程序退出
				os.Exit(1)
				return
			}
			fmt.Printf("✅ [SQL] 数据库模块 数据库文件创建成功！%s\n", env.Get("DB_NAME"))
		}
		// SQLite 配置
		dsn = fmt.Sprintf("file:%s?mode=rwc&cache=shared", env.Get("DB_NAME"))
		driver = "sqlite"
	case "pgsql", "postgres", "postgresql":
		// pgsql 配置
		dsn = fmt.Sprintf("%s:%s@%s:%s/%s?sslmode=disable", env.Get("DB_USER"), env.Get("DB_PASSWORD"), env.Get("DB_HOST"), env.Get("DB_PORT"), env.Get("DB_NAME"))
		driver = "pgx"
	case "mysql", "mariadb":
		// mysql 配置
		dsn = fmt.Sprintf("%s:%s@%s:%s/%s?sslmode=disable", env.Get("DB_USER"), env.Get("DB_PASSWORD"), env.Get("DB_HOST"), env.Get("DB_PORT"), env.Get("DB_NAME"))
		driver = "mysql"
	default:
		// 无法识别 程序退出
		fmt.Printf("⚠️ [SQL] 数据库模块 配置错误！无法识别的数据库类型:%s\n", env.Get("DB_TYPE"))
		// 程序退出
		os.Exit(1)
		return
	}

	var err error
	DB, err = sql.Open(driver, dsn)
	if err != nil {
		fmt.Printf("⚠️ [SQL] 数据库模块 配置错误！err: %v\n", err)
	}

	// 间隔2s
	time.Sleep(2 * time.Second)

	// 开始无限循环重连
	for {

		err = DB.Ping()
		if err == nil {
			fmt.Printf("✅ [SQL] 数据库 连接成功！\n")
			break // 连上了，跳出循环
		}

		fmt.Printf("🔄 [SQL] 数据库连接失败: %v。 9秒后尝试重连...\n", err)
		time.Sleep(9 * time.Second)
		// fmt.Printf("❌ [SQL] 数据库 重连中...\n")
	}

	// 设置连接池（连上后再设置）
	DB.SetMaxOpenConns(100)
	DB.SetMaxIdleConns(20)

}
