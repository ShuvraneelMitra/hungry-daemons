package main

import (
	"time"
	"github.com/ShuvraneelMitra/hungry-daemons/world"
)

func main() {
	ticker := time.NewTicker(time.Second / 2)

	w := world.NewWorld(
		"../config.toml",
		ticker,
	)

	w.Initialize()
	w.Run(10)

	<-w.Done()
}
