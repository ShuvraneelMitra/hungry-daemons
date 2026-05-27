package main

import (
	"sync"
	"context"

	"github.com/ShuvraneelMitra/hungry-daemons/gui"
	"github.com/ShuvraneelMitra/hungry-daemons/world"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}

	cfg := world.ParseConfig("../config.toml")
	earth, msgChannel := world.NewWorld(cfg)

	wg.Go(func(){
		earth.Initialize(cfg)
		earth.Run(ctx, cfg.Env.SimTicks)

		<-earth.Done()
	})

	gui.Run(msgChannel, cancel)
	wg.Wait()
}
