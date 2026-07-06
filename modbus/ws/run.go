package ws

import (
	"log"
	"net/http"

	web "modbus/chi"

	"github.com/go-chi/chi/v5"
)

// Run 将 /ws 挂载到 Chi 路由底座
func Run() {
	r := chi.NewRouter()
	r.Get("/", http.HandlerFunc(Handler))
	web.Mux.Mount("/ws", r)
	log.Print("✅ [WebSocket] 端点 /ws 已注册")
}
