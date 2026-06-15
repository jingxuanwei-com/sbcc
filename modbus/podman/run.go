package podman

// podman控制器模块
// 挂载路径："/podman"

import (
	"fmt"
	web "modbus/chi"
	"modbus/podman/home"

	"github.com/go-chi/chi/v5"
)

func Run() {

	r := chi.NewRouter()
	// ... 这里可以无脑复制官方文档的代码 ...

	r.Group(func(r chi.Router) {
		r.Get("/", home.Home)
		r.Get("/style.css", home.StaticHandler)
	})

	// 最后一炮打到web底座，搞定！
	web.Mux.Mount("/podman", r)
	fmt.Println("✅ [Podman] 模块 加载完成！")
}
