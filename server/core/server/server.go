// Package server web 服务模块：负责 web 服务的启动/重启/热重载（路由引擎在 web 模块）。
package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"server/core/web"
	"strings"
	"sync"
	"time"

	"github.com/1szt/go-pkg/env"
)

var (
	mu     sync.Mutex // 保护 server 和 lastListen（Restart 可从 HTTP handler 并发触发）
	server *http.Server
	// 上次成功启动的监听描述（地址+TLS 指纹），用于 Reload 判断配置是否变化
	lastListen string
)

type ServerStatus struct {
	IP     string `json:"ip"`
	Port   string `json:"port"`
	Status string `json:"status"`
}

// Start 启动 web 服务（阻塞，从配置读取 ip 和端口）。
// 热重载时 Restart 会关闭旧监听，导致 start 返回，但新监听已在 Restart 的 goroutine 中启动，
// 因此这里挂起防止 main 返回导致整个进程退出。
func Start() {
	start(addrFromConfig())
	select {}
}

// addrFromConfig 从配置读取 ip 和端口，拼出监听地址（端口缺省 9081）
func addrFromConfig() string {
	ip := env.Get("LFDD_SERVER_IP")
	port := env.Get("LFDD_PORT")
	if port == "" {
		port = "9081"
	}
	return ip + ":" + port
}

// tlsFromConfig 读取 HTTPS 配置，返回（是否启用, 证书路径, 密钥路径）。
// 优先使用手动配置的证书；未配置证书但开启 LFDD_SERVER_HTTPS_AUTO 时自动生成自签名证书。
// 以上都不满足即视为未启用，回退 HTTP。
func tlsFromConfig() (enabled bool, cert, key string) {
	if env.Get("LFDD_SERVER_HTTPS") != "true" {
		return false, "", ""
	}
	cert = env.Get("LFDD_SERVER_CERT")
	key = env.Get("LFDD_SERVER_KEY")
	if cert != "" && key != "" {
		return true, cert, key
	}
	if env.Get("LFDD_SERVER_HTTPS_AUTO") == "true" {
		cert, key, err := ensureSelfSignedCert()
		if err != nil {
			log.Printf("❌ [Server] 自签名证书生成失败: %v，回退 HTTP", err)
			return false, "", ""
		}
		return true, cert, key
	}
	log.Println("⚠️ [Server] 已开启 HTTPS 但证书/密钥未配置，回退 HTTP")
	return false, "", ""
}

// ensureSelfSignedCert 确保存在可用的自签名证书：已生成且未临近过期（30 天内）则复用，
// 否则生成 ECDSA P-256 证书（有效期 1 年）落盘到 data/tls/。
// SAN 包含 localhost、127.0.0.1、::1 及配置的服务器 IP，避免浏览器报域名不匹配。
func ensureSelfSignedCert() (certPath, keyPath string, err error) {
	certPath = filepath.Join("data", "tls", "selfsigned.crt")
	keyPath = filepath.Join("data", "tls", "selfsigned.key")

	// 已有证书且未临近过期 → 直接复用
	if certPEM, err := os.ReadFile(certPath); err == nil {
		if block, _ := pem.Decode(certPEM); block != nil {
			if c, perr := x509.ParseCertificate(block.Bytes); perr == nil && c.NotAfter.After(time.Now().Add(30*24*time.Hour)) {
				if _, kerr := os.Stat(keyPath); kerr == nil {
					return certPath, keyPath, nil
				}
			}
		}
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "lfdd"},
		NotBefore:    time.Now().Add(-time.Hour), // 容忍时钟偏差
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	if ip := net.ParseIP(env.Get("LFDD_SERVER_IP")); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", err
	}

	if err = os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return "", "", err
	}
	if err = os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644); err != nil {
		return "", "", err
	}
	if err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		return "", "", err
	}

	log.Println("🔐 [Server] 已生成自签名证书 data/tls/（浏览器会提示不受信任，属正常现象）")
	return certPath, keyPath, nil
}

// listenDesc 组合地址与 TLS 配置的描述，用于 Reload 对比配置是否变化
func listenDesc(addr string, tlsOn bool, cert, key string) string {
	if tlsOn {
		return addr + "|tls|" + cert + "|" + key
	}
	return addr
}

// start 在指定地址启动服务
func start(addr string) {
	srv := &http.Server{
		Addr:         addr,
		Handler:      web.Engine,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	tlsOn, cert, key := tlsFromConfig()

	mu.Lock()
	server = srv
	lastListen = listenDesc(addr, tlsOn, cert, key)
	mu.Unlock()

	var err error
	if tlsOn {
		log.Printf("🔒 [Server] 访问 https://%s", displayAddr(addr))
		err = srv.ListenAndServeTLS(cert, key)
	} else {
		log.Printf("🌐 [Server] 访问 http://%s", displayAddr(addr))
		err = srv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ [Server] 致命错误：%v", err)
	}
}

// displayAddr 把裸端口监听地址（如 ":9081"）转成可点击的 localhost 地址
func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

// Restart 停掉当前服务，从配置重新读取 ip 和端口重启
func Restart() {
	addr := addrFromConfig()

	mu.Lock()
	old := server
	mu.Unlock()

	if old != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := old.Shutdown(ctx); err != nil {
			log.Printf("❌ [Server] 停止旧服务失败: %v", err)
		} else {
			log.Printf("🔄 [Server] 已停止旧端口服务")
		}
	}

	// 在新地址启动（开一个 goroutine 避免阻塞）
	go start(addr)
	log.Printf("✅ [Server] 服务已切换到 %s", addr)
}

// Reload 智能重连（配置无变化且服务运行中则跳过，否则重启）
func Reload() {
	addr := addrFromConfig()
	tlsOn, cert, key := tlsFromConfig()

	mu.Lock()
	running := server != nil
	same := running && listenDesc(addr, tlsOn, cert, key) == lastListen
	mu.Unlock()

	// 配置没变且运行中 → 跳过
	if same {
		log.Println("🔄 [Server] 配置无变化且服务运行中，跳过重载")
		return
	}

	// 重载
	if !running {
		log.Println("🔄 [Server] 服务未运行，启动服务...")
	} else {
		log.Println("🔄 [Server] 配置已变化，重启服务...")
	}
	Restart()
}

// Status 获取当前服务状态
func Status() ServerStatus {
	mu.Lock()
	srv := server
	desc := lastListen
	mu.Unlock()

	if srv == nil {
		return ServerStatus{Status: "stopped"}
	}

	// 监听描述的首段是地址（TLS 部分以 | 分隔，去掉）
	addr := desc
	if i := strings.Index(addr, "|"); i >= 0 {
		addr = addr[:i]
	}

	// 从监听地址解析 ip 和端口
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("⚠️ [Server] 解析监听地址 %s 失败: %v", addr, err)
	}

	return ServerStatus{
		IP:     ip,
		Port:   port,
		Status: "running",
	}
}
