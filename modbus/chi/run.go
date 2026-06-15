package chi

// Web 引擎启动
// 使用 chi 路由库

import (
	"fmt"
	"modbus/env"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

var Mux = chi.NewRouter()

func Run() {

	// 一次性初始化配置文件
	env.Init([][]string{
		{"WEB_PORT", "9081", "端口"},
	})

	// 全局中间件设置

	// 捕获所有内部 panic
	Mux.Use(middleware.Recoverer)
	// 记录日志
	Mux.Use(middleware.Logger)
	// 限制每个IP每分钟最多100个请求
	Mux.Use(httprate.LimitByIP(100, 1*time.Minute))

	// 创建错误探测信封
	errChan := make(chan error, 1)

	// 协程启动
	go func() {
		if err := http.ListenAndServe(":"+env.Get("WEB_PORT"), Mux); err != nil {
			errChan <- err
		}
	}()

	// 【核心处理】检查是否报错
	select {
	case err := <-errChan:
		// 如果通道里有错，说明端口启动失败（如 Address already in use）
		fmt.Printf("❌ [Web] 致命错误：端口可能被占用或权限不足 | %v\n", err)
		os.Exit(1)
		return
	case <-time.After(100 * time.Millisecond):
		// 100ms 过去了没报错，说明端口占领成功
		fmt.Printf("✅ [Web] %s端口占领成功，底座已就绪\n", env.Get("WEB_PORT"))
		fmt.Printf("🌐 [Web] 访问 http://localhost:%s\n", env.Get("WEB_PORT"))
	}

}
