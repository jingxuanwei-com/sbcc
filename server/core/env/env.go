package env

import (
	"fmt"

	"github.com/1szt/go-pkg/env"
)

// Start 初始化配置文件并补全缺失的配置项（不覆盖已有值）。
// 配置文件默认路径为 data/.env，可用环境变量覆盖任意项。
func Start() {
	env.Path("data/.env")

	env.Add("SBCC_PORT", "9081", "端口")
	env.Add("SBCC_SERVER_IP", "", "服务器 IP")

	env.Add("SBCC_DEBUG", "false", "调试模式（true 时 gin 输出路由注册等调试日志，false 静默）")

	env.Add("SBCC_LOG_PATH", "data/log/sbcc.log", "日志文件路径（空则使用默认 data/log/sbcc.log）")

	env.Add("SBCC_SERVER_HTTPS", "false", "是否启用 HTTPS（true/false）")
	env.Add("SBCC_SERVER_HTTPS_AUTO", "true", "是否自签名证书（true/false）")
	env.Add("SBCC_SERVER_CERT", "", "HTTPS 证书路径")
	env.Add("SBCC_SERVER_KEY", "", "HTTPS 私钥路径")
	env.Add("SBCC_SERVER_INSTALL", "true", "是否已完成初始化安装（true/false）")
	env.Add("SBCC_CORS_ORIGINS", "*", "跨域允许的来源（* 表示全部，多个用逗号分隔），前后端分域部署时配置")

	env.Add("SBCC_DB_TYPE", "sqlite", "数据库类型（sqlite/pgsql/mysql）")
	env.Add("SBCC_DB_PATH", "data/db/sbcc.db", "SQLite 数据库文件路径")
	env.Add("SBCC_DB_HOST", "", "数据库主机地址")
	env.Add("SBCC_DB_PORT", "", "数据库端口")
	env.Add("SBCC_DB_NAME", "", "数据库名")
	env.Add("SBCC_DB_USER", "", "数据库用户名")
	env.Add("SBCC_DB_PASSWORD", "", "数据库密码")

	env.Add("SBCC_UPLOAD_PATH", "data/upload", "图片/视频上传存储目录")

	fmt.Println("✅ [Env] 配置文件 初始化完成")
}
