package home

// 依赖模块 ：web
// 主页模块负责处理用户访问根路径的请求
// 挂载路径："/"

import (
	"log"
	web "modbus/chi"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Run() {

	r := chi.NewRouter()
	// ... 这里可以无脑复制官方文档的代码 ...

	r.Group(func(r chi.Router) {
		r.Get("/", home)
		r.Get("/data", handleData)
	})

	// 最后一炮打到web底座，搞定！
	web.Mux.Mount("/", r)
	log.Print("✅ [Home] 主页模块 加载完成！")
}

func home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from Home!"))
}
