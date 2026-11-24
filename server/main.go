package main

import (
	"server/core/env"
	"server/core/motd"
)

func main() {
	motd.Start()
	env.Start()

}
