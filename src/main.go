package main

import (
	"sync"
	"time"

	"github.com/ShuvraneelMitra/hungry-daemons/gui"
	"github.com/ShuvraneelMitra/hungry-daemons/world"
)

func main() {
	ticker := time.NewTicker(time.Second / 2)

	s := sync.WaitGroup{}

	s.Go(func(){
		w := world.NewWorld(
			"../config.toml",
			ticker,
		)

		w.Initialize()
		w.Run(100)

		<-w.Done()
	})

	gui.Run()
	s.Wait()
}
