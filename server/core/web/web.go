// Package web web 框架初始化模块：持有全局 Engine 并注册全局中间件，必须在模块注册路由前调用。
package web

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"server/core/config"
)

// Engine 全局路由引擎，各业务模块通过它注册自己的路由
var Engine *gin.Engine

// Start 注册全局中间件（Logger + Recovery + 初始化检查）
// 必须在所有业务模块注册路由之前调用（gin 中间件对之后注册的路由才生效）
func Start() {
	// 非调试模式切换 gin 为 release，屏蔽路由注册等调试日志刷屏
	if !config.Cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	Engine = gin.New()

	Engine.Use(gin.Logger())
	Engine.Use(gin.Recovery())
	// 跨域要放在鉴权之前：预检 OPTIONS 请求不带身份，需直接放行

	fmt.Println("✅ [Web]核心初始化 加载完成！")
}
