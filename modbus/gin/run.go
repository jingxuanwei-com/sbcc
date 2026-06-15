package gin

import (
	"fmt"
	web "modbus/chi"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Run() {

	// 最后一炮打到web底座，搞定！
	web.Mux.Mount("/gin", test())
	fmt.Println("🎉 [Gin] Gin测试模块 加载完成！")
}

// Gin测试模块 使用Gin框处理器
func test() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.GET("/gin", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "We love Gin!"})
	})
	return g // 返回这个 Gin 实例作为 http.Handler
}
