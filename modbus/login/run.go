// ============================================================
//
//	登录认证模块 — SBCC 控制中心
//	功能：用户登录/注销/状态查询，使用 Scs 管理 Session
//	依赖：gorm、scs
//	启动方式：由 modbus/main 模块统一调用
//
// ============================================================
package login

import (
	"encoding/json"
	"log"
	"net/http"

	web "modbus/chi"
	"modbus/gorm"
	"modbus/scs"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

// --- 请求/响应结构体 ---

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type meResponse struct {
	LoggedIn bool   `json:"logged_in"`
	Username string `json:"username,omitempty"`
}

// ═══════════════════════════════════════════════════════════════
//
//	Run — 登录模块启动入口
//
//	在 main/run.go 里调用：
//	  gorm.Run()
//	  scs.Run()
//	  login.Run()   ← 必须在 gorm.Run() 和 scs.Run() 之后
//
//	注册路由：
//	  POST /api/login       — 登录
//	  POST /api/login/logout — 注销
//	  GET  /api/login/me    — 查看当前登录状态
//
// ═══════════════════════════════════════════════════════════════
func Run() {
	// 初始化数据库表
	InitDB()

	r := chi.NewRouter()

	// 所有 /api/login/* 路由启用 Session 中间件
	r.Group(func(r chi.Router) {
		r.Use(scs.Scs.LoadAndSave)

		r.Post("/", handleLogin)
		r.Post("/logout", handleLogout)
		r.Get("/me", handleMe)
	})

	web.Mux.Mount("/api/login", r)
	log.Print("✅ [Login] 登录认证模块 加载完成！")
}

// ─── handler: 登录 ─────────────────────────────────────────
func handleLogin(w http.ResponseWriter, r *http.Request) {
	// 1. 解析 JSON 请求体
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, loginResponse{
			Ok: false, Message: "请求数据格式错误",
		})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, loginResponse{
			Ok: false, Message: "用户名和密码不能为空",
		})
		return
	}

	// 2. 从数据库查询用户
	var user User
	err := gorm.DB.Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, loginResponse{
			Ok: false, Message: "用户名或密码错误",
		})
		return
	}

	// 3. 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, loginResponse{
			Ok: false, Message: "用户名或密码错误",
		})
		return
	}

	// 4. 登成功 → 续命 Session token（防 Session 固定攻击）
	if err := scs.Scs.RenewToken(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, loginResponse{
			Ok: false, Message: "服务器内部错误",
		})
		return
	}

	// 5. 写入 Session
	scs.Scs.Put(r.Context(), "user_id", user.ID)
	scs.Scs.Put(r.Context(), "username", user.Username)

	writeJSON(w, http.StatusOK, loginResponse{
		Ok: true, Message: "登录成功",
	})
}

// ─── handler: 注销 ─────────────────────────────────────────
func handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := scs.Scs.Destroy(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, loginResponse{
			Ok: false, Message: "注销失败",
		})
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Ok: true, Message: "已注销",
	})
}

// ─── handler: 查看登录状态 ────────────────────────────────
func handleMe(w http.ResponseWriter, r *http.Request) {
	userID := scs.Scs.GetInt64(r.Context(), "user_id")
	username := scs.Scs.GetString(r.Context(), "username")

	if userID == 0 {
		writeJSON(w, http.StatusOK, meResponse{LoggedIn: false})
		return
	}

	writeJSON(w, http.StatusOK, meResponse{
		LoggedIn: true,
		Username: username,
	})
}

// ─── 辅助函数 ─────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// RequireAuth 中间件：未登录则返回 401
// 用法：
//
//	r.Group(func(r chi.Router) {
//	    r.Use(login.RequireAuth)
//	    r.Get("/protected", myHandler)
//	})
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := scs.Scs.GetInt64(r.Context(), "user_id")
		if userID == 0 {
			writeJSON(w, http.StatusUnauthorized, loginResponse{
				Ok: false, Message: "请先登录",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
