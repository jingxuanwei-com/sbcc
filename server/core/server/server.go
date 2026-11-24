// Package server web 服务模块：按配置启动 HTTP/HTTPS 服务（路由引擎在 web 模块）。
package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"server/core/config"
	"server/core/web"
)

// Start 按 data/config.yaml 的 server 配置启动 web 服务，阻塞运行。
// 开启 https 且配置了 cert_file/key_file 时走 HTTPS，否则回退 HTTP。
func Start() {
	s := config.Cfg.Server

	port := s.Port
	if port == 0 {
		port = 9081
	}
	addr := s.IP + ":" + strconv.Itoa(port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      web.Engine,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if s.HTTPS && s.CertFile != "" && s.KeyFile != "" {
		log.Printf("🔒 [Server] 访问 https://%s", displayAddr(addr))
		if err := srv.ListenAndServeTLS(s.CertFile, s.KeyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ [Server] 致命错误：%v", err)
		}
		return
	}

	if s.HTTPS {
		log.Println("⚠️ [Server] 已开启 https 但未配置 cert_file/key_file，回退 HTTP")
	}
	log.Printf("🌐 [Server] 访问 http://%s", displayAddr(addr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ [Server] 致命错误：%v", err)
	}
}

// displayAddr 把裸端口监听地址（如 ":9081"）转成可读的 localhost 地址
func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}
