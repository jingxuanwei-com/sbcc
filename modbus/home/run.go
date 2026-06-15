package home

// 依赖模块 ：web
// 主页模块负责处理用户访问根路径的请求
// 挂载路径："/"

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	web "modbus/chi"
	"net/http"
)

func Run() {

	r := chi.NewRouter()
	// ... 这里可以无脑复制官方文档的代码 ...

	r.Group(func(r chi.Router) {
		r.Get("/", home)
	})

	// 最后一炮打到web底座，搞定！
	web.Mux.Mount("/", r)
	fmt.Println("✅ [Home] 主页模块 加载完成！")
}

func home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from Home!"))
}
