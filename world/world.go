package world

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	. "github.com/ShuvraneelMitra/hungry-daemons/managers"
	"github.com/pelletier/go-toml/v2"
)

const(
	ID_LENGTH  = 10
	DEATH_PROB = 0.4 // Probability that at the current ticker if age > lifeExpectancy the organism dies
) 

type EventHandler func(world *World, data any)

var (
	once, initializeOnce sync.Once
	cfg struct {
		Env struct{
			InitPop   int `toml:"initial_population"`
			MaxPop    int `toml:"max_population"`
			SimTicks  int `toml:"simulation_ticks"`
			MaxCPU    int `toml:"max_cpu_tokens"`
			LifeExp   int `toml:"life_expectancy"`
			TicksPerS int `toml:"ticks_per_s"`
		} `toml:"env"`
	}
	handlerTable = map[EventType]EventHandler{
		KILL: func(world *World, data any) {
				info, ok := data.([]interface{})
				if !ok {
					return 
				}
				world.Kill(info[0].(string), info[1].(DeathType))
			},
		RELEASE_CPU : func(world *World, data any) {
						daemonId, ok := data.(string)
						if !ok {
							return 
						}
						world.ReleaseCpu(daemonId)
					},
		SPAWN : func(world *World, data any) {
					genome, ok := data.(Genome)
					if !ok { return }
					world.Spawn(genome, world.CurrentTick())
				},

		REQUEST_CPU : func(world *World, data any) {
						genome, ok := data.([]any)
						if !ok { return }

						id, tokens := genome[0].(string), genome[1].(int)
						world.AllocateTokens(id, tokens)
					},
	}
)



type Environment interface {
	CurrentTick() int
	SendSignal(signal EventType, data any)
	TicksPerS() int 
	GetPopulation() int
}

type World struct {
	numOrganisms   int
	numFreeTokens  int
	lifeExpectancy int
	ticksPerS      int
	currentTick    int
	maxPop         int

    organisms      map[string]*Daemon
	mtx            *sync.RWMutex
	wg             *sync.WaitGroup
	eventMgr       *EventManager
	ticker         *time.Ticker
	reqChannel     chan Event
	ctx            context.Context
	cancelFunc     context.CancelFunc
	shutdownOnce   sync.Once
	done           chan struct{}

	stats          *Stats
}

//------------------------ THREAD-SAFETY -----------------------------

type organismSnapshot struct {
	id     string
	daemon *Daemon
	age    int
	lastHeldTokens int
	lifeWithoutFood int
	minimumLifespan int
}

func (world *World) snapshotOrganisms() []organismSnapshot {
	world.mtx.RLock()
	daemons := make(map[string]*Daemon, len(world.organisms))
	for id, daemon := range world.organisms {
		daemons[id] = daemon
	}
	world.mtx.RUnlock()

	snapshot := make([]organismSnapshot, 0, len(daemons))
	for id, daemon := range daemons {
		_, lastHeld := daemon.State()
		snapshot = append(snapshot, organismSnapshot{
			id:              id,
			daemon:          daemon,
			age:             daemon.Age(),
			lastHeldTokens:  lastHeld,
			lifeWithoutFood: daemon.Genome.LifeWithoutFood,
			minimumLifespan: daemon.Genome.MinimumLifespan,
		})
	}

	return snapshot
}

//----------------- MOST IMPORTANT FUNCTIONS -------------------------

func NewWorld(configFile string, ticker *time.Ticker) *World {
	once.Do(func(){
		content, err := os.ReadFile(configFile)
		if err != nil {
			panic("Error reading file: " + err.Error())
		}

		if err := toml.Unmarshal(content, &cfg); err != nil {
			panic("Error parsing TOML: " + err.Error())
		}
	})

	if(cfg.Env.TicksPerS == 0) {
		panic("ticks_per_second == 0 in config file, world stopped!")
	}

	ctx, cancel := context.WithCancel(context.Background())

	world := &World{
		numOrganisms  : 0,
		numFreeTokens : cfg.Env.MaxCPU,
		lifeExpectancy: cfg.Env.LifeExp,
		ticksPerS     : cfg.Env.TicksPerS,
		currentTick   : 0,
		maxPop        : cfg.Env.MaxPop,

		mtx           : &sync.RWMutex{},
		wg            : &sync.WaitGroup{},

		eventMgr      : NewEventManager(),
		organisms     : make(map[string]*Daemon),
		ticker        : ticker,
		reqChannel    : make(chan Event, cfg.Env.MaxPop + 1),
		ctx           : ctx,
		cancelFunc    : cancel,
		shutdownOnce  : sync.Once{},
		done          : make(chan struct{}),

		stats         : CreateStats(),
	}

	// SUBSCRIBE TO EVENTS
	world.eventMgr.Subscribe(KILL, world.reqChannel)
	world.eventMgr.Subscribe(RELEASE_CPU, world.reqChannel)
	world.eventMgr.Subscribe(SPAWN, world.reqChannel)
	world.eventMgr.Subscribe(REQUEST_CPU, world.reqChannel)
	return world
}

func (world *World) Initialize(){
	initializeOnce.Do(
		func(){	
			world.mtx.Lock()
			defer world.mtx.Unlock()
			
			world.numOrganisms = cfg.Env.InitPop
			/*
			Why do this here and not in NewWorld()? Suppose you call 
			NewWorld() but forget to call Initialize(). Then the World
			object will have numOrganisms > 0 but an empty organisms map.
			*/
			
			for {
				id := generateRandomString(ID_LENGTH)
				surname := generateRandomString(ID_LENGTH)
				if _, exists := world.organisms[id]; exists {
					continue
				}
				
				genome := Genome{
					ID               : id,
					Surname          : surname,
					ReplicationRate  : rand.Float64(),
					CPUHunger        : rand.IntN(cfg.Env.MaxCPU),
					MutationChance   : rand.Float64(),
					LifeWithoutFood  : rand.IntN(world.lifeExpectancy) + 1,
					MinimumLifespan  : rand.IntN(world.lifeExpectancy),
					MinimumHoldTime  : rand.IntN(world.lifeExpectancy),
					InstructionSet   : nil,
				}
				world.organisms[id] = &Daemon{
					Genome        : genome,
					CurrentTokens : 0,
					CreatedTick   : world.currentTick,
					LastHeldTokens: -1,
					 
				    mtx           : world.mtx,

					Env           : world,
					Channel       : make(chan Event),
					TickC 		  : make(chan int, 1),
				}

				world.stats.TrackBirth(world.organisms[id].Genome.Surname)	

				if len(world.organisms) == world.numOrganisms {
					break
				}
			}
		})
}

func (world *World) Run(simTicks int) {
	go func() {
		for _, daemon := range world.organisms {
			temp_daemon := daemon
			world.wg.Go(func() { temp_daemon.Run(world.ctx) })
		}

		for {
			select {
				case <-world.ctx.Done():
					world.Shutdown()
					return

				case <-world.ticker.C:
					tickCtx, cancel := context.WithTimeout(world.ctx, time.Second/time.Duration(world.ticksPerS))
					world.Tick(tickCtx)
					cancel()

					if world.currentTick >= simTicks {
						world.Shutdown()
						return 
					}
			}
		}
	}()
}

// -------------------- ALL THE HELPER/FACTORY FUNCTIONS ------------------

func (world *World) AllocateTokens(daemonId string, tokens int) int {
	world.mtx.Lock()

	if world.numFreeTokens < tokens {
		world.mtx.Unlock()
		return -1
	}

	daemon, ok := world.organisms[daemonId]
	if !ok {
		world.mtx.Unlock()
		return -1
	}

	world.numFreeTokens -= tokens
	currentTick := world.currentTick
	world.mtx.Unlock()

	daemon.SetTokens(tokens, currentTick)
	return 0
}

func (world *World) broadcastTick(tick int) {
	world.mtx.RLock()
	defer world.mtx.RUnlock()
	defer func() {
		recover()
	}()

	for _, daemon := range world.organisms {
		daemon.TickC <- tick
	}
}

func (world *World) CurrentTick() int {
	world.mtx.RLock()
	defer world.mtx.RUnlock()
	return world.currentTick
}

func (world *World) Done() <-chan struct{} {
	return world.done
}

func (world *World) GetOrganism(id string) (*Daemon, bool) {
	world.mtx.RLock()
	defer world.mtx.RUnlock()
	daemon, ok := world.organisms[id]
	return daemon, ok
}

func (world *World) GetPopulation() int {
	world.mtx.RLock()
	defer world.mtx.RUnlock()
	return world.numOrganisms
}

func (world *World) handleRequest(event Event) {
	if handler, ok := handlerTable[event.Name]; ok {
		handler(world, event.Data)
	}
}

func (world *World) Kill(daemonId string, reason DeathType) {
	world.mtx.Lock()

	daemon, ok := world.organisms[daemonId]
	if !ok {
		world.mtx.Unlock()
		return
	}

	world.numFreeTokens += daemon.CurrentTokens
	world.numOrganisms--
	delete(world.organisms, daemonId)
	world.mtx.Unlock()

	world.stats.TrackDeath(reason)

	close(daemon.Channel)
	close(daemon.TickC)
	daemon.ClearTokens(world.currentTick)
}

func (world *World) ReleaseCpu(daemonId string) {
	world.mtx.Lock()

	daemon, ok := world.organisms[daemonId]
	if !ok {
		world.mtx.Unlock()
		return
	}

	world.numFreeTokens += daemon.CurrentTokens
	world.mtx.Unlock()

	daemon.ClearTokens(world.currentTick)
	
}

func (world *World) SendSignal(signal EventType, data any) {
	world.eventMgr.Send(signal, data)
}

func (world *World) Shutdown(){
	world.shutdownOnce.Do(func(){
		defer world.ticker.Stop()

		world.cancelFunc()

		world.eventMgr.Unsubscribe(KILL, world.reqChannel)
		world.eventMgr.Unsubscribe(RELEASE_CPU, world.reqChannel)
		world.eventMgr.Unsubscribe(SPAWN, world.reqChannel)
		world.eventMgr.Unsubscribe(REQUEST_CPU, world.reqChannel)

		world.wg.Wait()

		close(world.reqChannel)
		close(world.done)
	})
}

func (world *World) Spawn(genome Genome, tick int) {
	world.mtx.Lock()

	if world.numOrganisms >= world.maxPop {
		world.mtx.Unlock()
		return 
	}

	daemon := NewDaemon(genome, world, tick)

	if _, exists := world.organisms[genome.ID]; exists {
		world.mtx.Unlock()
		return
	}

	world.organisms[daemon.Genome.ID] = daemon
	
	world.numOrganisms++
	world.mtx.Unlock()

	world.wg.Go(func() { daemon.Run(world.ctx) })
}

func (world *World) Tick(ctx context.Context) {
	world.mtx.Lock()
	world.currentTick++
	fmt.Println("Tick number ", world.currentTick)
	currentTick := world.currentTick
	population := world.numOrganisms
	world.mtx.Unlock()

	world.broadcastTick(currentTick)


	var wg sync.WaitGroup
	wg.Go(func() { // Killing in the name of
		organisms := world.snapshotOrganisms()

		for _, daemon := range organisms {
			if daemon.age > daemon.minimumLifespan {
				probablyExecute(DEATH_PROB, func(){
					world.eventMgr.Send(KILL, []any{
						daemon.id, DeathAge,
					})
				})
			} else if currentTick - daemon.lastHeldTokens > daemon.lifeWithoutFood {
				world.eventMgr.Send(KILL, []any{
						daemon.id, DeathStarvation,
				})
			}
		}

		if population > world.maxPop {
			var oldestId string
			var oldestAge int = 0
			for _, daemon := range organisms {
				if daemon.age > oldestAge {
					oldestAge = daemon.age
					oldestId = daemon.id
				}
			}
			world.eventMgr.Send(KILL, []any{
				oldestId, DeathCulling,
			})
		}
	})

	for {
		select {
			case <-ctx.Done():
				goto done
			case event, ok := <-world.reqChannel:
				if !ok {
					goto done
				}
				world.handleRequest(event)
		}
	}

	done:
		world.stats.UpdateMetrics()
		wg.Wait()
}

func (world *World) TicksPerS() int {
	return world.ticksPerS
}




