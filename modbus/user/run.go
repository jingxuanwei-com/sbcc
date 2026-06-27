// ============================================================
//
//	用户模块 — SBCC 控制中心
//	功能：登录/注销/注册/状态查询，管理 users 表
//	依赖：gorm、scs
//	启动方式：由 modbus/main 模块统一调用
//
// ============================================================
package user

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

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type apiResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type meResponse struct {
	LoggedIn bool   `json:"logged_in"`
	Username string `json:"username,omitempty"`
}

// ═══════════════════════════════════════════════════════════════
//
//	Run — 用户模块启动入口
//
//	在 main/run.go 里调用：
//	  gorm.Run()
//	  scs.Run()
//	  user.Run()   ← 必须在 gorm.Run() 和 scs.Run() 之后
//
//	注册路由：
//	  POST /api/user            — 登录
//	  POST /api/user/logout     — 注销
//	  GET  /api/user/me         — 查看当前登录状态
//	  POST /api/user/register   — 注册新用户
//
// ═══════════════════════════════════════════════════════════════
func Run() {
	// 初始化数据库表
	InitDB()

	r := chi.NewRouter()

	// 所有 /api/user/* 路由启用 Session 中间件
	r.Group(func(r chi.Router) {
		r.Use(scs.Scs.LoadAndSave)

		r.Post("/", handleLogin)
		r.Post("/logout", handleLogout)
		r.Get("/me", handleMe)
		r.Post("/register", handleRegister)
	})

	web.Mux.Mount("/api/user", r)
	log.Print("✅ [User] 用户模块 加载完成！")
}

// ─── handler: 登录 ─────────────────────────────────────────
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{
			Ok: false, Message: "请求数据格式错误",
		})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{
			Ok: false, Message: "用户名和密码不能为空",
		})
		return
	}

	// 从数据库查询用户
	var user User
	err := gorm.DB.Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiResponse{
			Ok: false, Message: "用户名或密码错误",
		})
		return
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, apiResponse{
			Ok: false, Message: "用户名或密码错误",
		})
		return
	}

	// 续命 Session token（防 Session 固定攻击）
	if err := scs.Scs.RenewToken(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{
			Ok: false, Message: "服务器内部错误",
		})
		return
	}

	// 写入 Session
	scs.Scs.Put(r.Context(), "user_id", user.ID)
	scs.Scs.Put(r.Context(), "username", user.Username)

	writeJSON(w, http.StatusOK, apiResponse{
		Ok: true, Message: "登录成功",
	})
}

// ─── handler: 注销 ─────────────────────────────────────────
func handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := scs.Scs.Destroy(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{
			Ok: false, Message: "注销失败",
		})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{
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

// ─── handler: 注册 ─────────────────────────────────────────
func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{
			Ok: false, Message: "请求数据格式错误",
		})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{
			Ok: false, Message: "用户名和密码不能为空",
		})
		return
	}

	if len(req.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, apiResponse{
			Ok: false, Message: "密码长度不能少于6位",
		})
		return
	}

	// 检查用户名是否已存在
	var count int64
	gorm.DB.Model(&User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		writeJSON(w, http.StatusConflict, apiResponse{
			Ok: false, Message: "用户名已存在",
		})
		return
	}

	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{
			Ok: false, Message: "服务器内部错误",
		})
		return
	}

	// 创建用户
	user := User{
		Username:     req.Username,
		PasswordHash: string(hash),
	}
	if err := gorm.DB.Create(&user).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{
			Ok: false, Message: "注册失败，请稍后重试",
		})
		return
	}

	// 注册成功后自动登录
	if err := scs.Scs.RenewToken(r.Context()); err == nil {
		scs.Scs.Put(r.Context(), "user_id", user.ID)
		scs.Scs.Put(r.Context(), "username", user.Username)
	}

	writeJSON(w, http.StatusCreated, apiResponse{
		Ok: true, Message: "注册成功",
	})
}

// ─── 辅助函数 ─────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// RequireLogin 中间件：未登录则返回 401
// 用法：
//
//	r.Group(func(r chi.Router) {
//	    r.Use(user.RequireLogin)
//	    r.Get("/protected", myHandler)
//	})
func RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := scs.Scs.GetInt64(r.Context(), "user_id")
		if userID == 0 {
			writeJSON(w, http.StatusUnauthorized, apiResponse{
				Ok: false, Message: "请先登录",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
