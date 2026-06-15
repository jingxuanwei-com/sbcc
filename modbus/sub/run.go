package sub

// 依赖模块 ：chi
// 订阅api模块
// 订阅api模块负责处理用户访问订阅api路径的请求
// 挂载路径："/api/sub"

import (
	"fmt"
	web "modbus/chi"
	"modbus/gorm"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func Run() {
	// 初始化数据库
	InitDB()

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Get("/", sub)
		r.Get("/node", subNode)
	})

	web.Mux.Mount("/api/sub", r)
	fmt.Println("✅ [Sub] 订阅api模块 加载完成！")

}

func sub(w http.ResponseWriter, r *http.Request) {

	// 1. 获取原始 UA
	rawUA := r.UserAgent()

	// 2. 转换为小写，确保 Clash, clash, CLASH 都能匹配
	ua := strings.ToLower(rawUA)

	// 3. 调试大法：直接在控制台看一眼到底传了什么
	// fmt.Printf("【调试】当前请求 UA: %s\n", rawUA)

	// 4. 核心逻辑：只要包含 "clash" 字符串就放行
	if !strings.Contains(ua, "clash") {
		// 如果不包含，返回 404
		http.NotFound(w, r)
		return
	}

	// 获取?token参数
	token := r.URL.Query().Get("token")

	// 对比数据库是否存在该token
	// 从数据库查询token是否存在
	var sub Sub

	err := gorm.DB.Where("token = ?", token).First(&sub).Error
	if err != nil {
		// 返回404错误页面
		http.NotFound(w, r)
		return
	}

	ConfigPath := "data/clash.yaml"

	absPath, _ := filepath.Abs(ConfigPath)

	info, err := os.Stat(absPath)
	if err != nil || info.Size() == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "错误：配置文件不存在或为空。路径：%s", absPath)
		return
	}

	// 配置文件显示名称
	filename := "SBCC"
	// 更新间隔（单位: 小时）
	update := "1"
	// 流量信息(单位: 字节)

	// 使用int64类型表示流量信息，避免溢出
	total := int64(1024 * 1024 * 1024 * 222)
	upload := int64(1024 * 1024 * 222)
	download := int64(1024 * 1024 * 500)
	// 到期时间(时间戳)
	expire := time.Date(2077, 12, 10, 0, 0, 0, 0, time.Local).Unix()
	weburl := "http://localhost:9081"

	// Header 文档
	//https://www.clashverge.dev/guide/url_schemes.html

	// 配置文件名
	w.Header().Set("content-disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(filename)))

	// 更新间隔
	w.Header().Set("profile-update-interval", update)

	// 流量信息(单位: 字节)、到期信息(时间戳)
	w.Header().Set("subscription-userinfo", fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d", upload, download, total, expire))

	// 首页URL
	w.Header().Set("profile-web-page-url", weburl)

	// 基础 Header 设置
	w.Header().Set("content-type", "application/x-yaml; charset=utf-8")

	// 发送文件
	http.ServeFile(w, r, absPath)

}

func subNode(w http.ResponseWriter, r *http.Request) {
	// 获取?token参数
	token := r.URL.Query().Get("token")

	// 1. 校验 Token
	if token != "caj8r79cejyrl3gph4cs5c5ox6cjdph1" {
		// Token 不对，直接报 403 Forbidden 或 401
		http.Error(w, "Unauthorized access", http.StatusUnauthorized)
		return
	}

	ConfigPath := "data/1szt.yaml"

	absPath, _ := filepath.Abs(ConfigPath)

	info, err := os.Stat(absPath)
	if err != nil || info.Size() == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "错误：配置文件不存在或为空。路径：%s", absPath)
		return
	}

	// 4. 返回文件
	// http.ServeFile 会自动处理 Content-Type 和绝对路径转换
	http.ServeFile(w, r, absPath)
}
