package main

import (
	"server/core/config"
	"server/core/motd"
	"server/core/server"
	"server/core/web"
)

func main() {
	motd.Start()
	config.Start()
	web.Start()
	server.Start()

}
