// ============================================================
//
//	Session 管理模块 — SBCC 控制中心
//	功能：提供 HTTP Session 管理，使用公共 sql.DB 作为存储后端
//	依赖：github.com/alexedwards/scs/v2
//	启动方式：由 modbus/main 模块统一调用
//
// ============================================================
package scs

import (
	"log"
	"strconv"
	"time"

	"modbus/env"
	dbsql "modbus/sql"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

// Scs 全局 Session 管理器，供其他模块使用
var Scs *scs.SessionManager

// ═══════════════════════════════════════════════════════════════
//
//	Run — Session 模块启动入口
//
//	在 main/run.go 里调用：
//	  sql.Run()
//	  scs.Run()   ← 必须在 sql.Run() 之后调用
//	  home.Run()
//
//	做了三件事：
//	  1. 初始化 Session 配置（过期时间等）
//	  2. 根据数据库类型选择对应的 SCS 存储后端
//	  3. 暴露 Scs 供其他模块使用
//
// ═══════════════════════════════════════════════════════════════
func Run() {
	// ─── 1. 初始化配置 ──────────────────────────────────────
	env.Init([][]string{
		{"SESSION_LIFETIME", "86400", "Session 过期时间（秒），默认 24 小时"},
		{"SESSION_IDLE_TIMEOUT", "0", "Session 空闲超时（秒），0=不限制"},
	})

	// ─── 2. 创建 Session 管理器 ────────────────────────────
	Scs = scs.New()

	lifetime, _ := strconv.Atoi(env.Get("SESSION_LIFETIME"))
	Scs.Lifetime = time.Duration(lifetime) * time.Second

	if idle, _ := strconv.Atoi(env.Get("SESSION_IDLE_TIMEOUT")); idle > 0 {
		Scs.IdleTimeout = time.Duration(idle) * time.Second
	}

	// ─── 3. 选择存储后端 ──────────────────────────────────
	switch env.Get("DB_TYPE") {
	case "sqlite", "sqllite", "sqlite3":
		Scs.Store = sqlite3store.New(dbsql.DB)
	case "pgsql", "postgres", "postgresql":
		Scs.Store = postgresstore.New(dbsql.DB)
	case "mysql", "mariadb":
		Scs.Store = mysqlstore.New(dbsql.DB)
	default:
		log.Fatalf("⚠️ [Scs] 不支持的数据库类型: %s", env.Get("DB_TYPE"))
		return
	}

	log.Print("✅ [Scs] Session 管理模块 加载完成！")
}
