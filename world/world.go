package world

import (
	"context"
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
				daemonId, ok := data.(string)
				if !ok {
					return 
				}
				world.Kill(daemonId)
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
					world.Spawn(genome, world.currentTick)
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
	mtx            sync.RWMutex
	eventMgr       *EventManager
	ticker         *time.Ticker
	reqChannel     chan Event
	ctx            context.Context
	cancelFunc     context.CancelFunc
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

		eventMgr      : NewEventManager(),
		organisms     : make(map[string]*Daemon),
		ticker        : ticker,
		reqChannel    : make(chan Event),
		ctx           : ctx,
		cancelFunc    : cancel,
	}
	world.eventMgr.Subscribe(KILL, world.reqChannel)
	world.eventMgr.Subscribe(RELEASE_CPU, world.reqChannel)
	world.eventMgr.Subscribe(SPAWN, world.reqChannel)

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
					MinimumLifespan  : rand.IntN(world.lifeExpectancy),
					InstructionSet   : nil,
				}
				world.organisms[id] = &Daemon{
					Genome        : genome,
					CurrentTokens : 0,
					CreatedTick   : world.currentTick,
					LastHeldTokens: -1,

					Env           : world,
					Channel       : make(chan Event),
				}
				

				if len(world.organisms) == world.numOrganisms {
					break
				}
			}
		})
}

func (world *World) Run(simTicks int) {
	go func() {
		defer world.ticker.Stop()

		for _, daemon := range world.organisms {
			go func() { daemon.Run(world.ctx, world.ticker) }()
		}

		for {
			select {
				case <-world.ctx.Done():
					return

				case <-world.ticker.C:
					tickCtx, cancel := context.WithTimeout(world.ctx, time.Second/time.Duration(world.ticksPerS))
					world.Tick(tickCtx)
					cancel()

					if world.currentTick >= simTicks {
						return 
					}
			}
		}
	}()
}

// -------------------- ALL THE HELPER/FACTORY FUNCTIONS ------------------

func (world *World) AllocateTokens(daemonId string, tokens int) int {
	if tokens <= 0 {
		return -1
	}

	world.mtx.Lock()
	defer world.mtx.Unlock()

	if world.numFreeTokens < tokens {
		return -1
	}

	daemon, ok := world.organisms[daemonId]
	if !ok {
		return -1
	}

	world.numFreeTokens -= tokens
	daemon.CurrentTokens += tokens
	daemon.LastHeldTokens = world.currentTick
	return 0
}

func (world *World) CurrentTick() int {
	world.mtx.RLock()
	defer world.mtx.RUnlock()
	return world.currentTick
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

func (world *World) Kill(daemonId string) {
	world.mtx.Lock()
	defer world.mtx.Unlock()

	daemon, ok := world.organisms[daemonId]
	if !ok {
		return
	}

	world.numFreeTokens += daemon.CurrentTokens
	world.numOrganisms--
	delete(world.organisms, daemonId)
	close(daemon.Channel)
}

func (world *World) ReleaseCpu(daemonId string) {
	world.mtx.Lock()
	defer world.mtx.Unlock()

	daemon, ok := world.organisms[daemonId]
	if !ok {
		return
	}

	world.numFreeTokens += daemon.CurrentTokens
	daemon.CurrentTokens = 0
	daemon.LastHeldTokens = world.currentTick
}

func (world *World) SendSignal(signal EventType, data any) {
	world.eventMgr.Send(signal, data)
}

func (world *World) Spawn(genome Genome, tick int) {
	daemon := NewDaemon(genome, world, tick)
	world.mtx.Lock()
	defer world.mtx.Unlock()

	if _, exists := world.organisms[genome.ID]; exists {
		return
	}

	world.organisms[daemon.Genome.ID] = daemon
	
	world.numOrganisms++
	go func() { daemon.Run(world.ctx, world.ticker) }()
}

func (world *World) Tick(ctx context.Context) {
	world.mtx.Lock()
	world.currentTick++
	world.mtx.Unlock()

	var wg sync.WaitGroup
	wg.Go(func() { // Killing in the name of
		for id, daemon := range world.organisms {
			if daemon.Age() > daemon.Genome.MinimumLifespan {
				probablyExecute(DEATH_PROB, func(){
					trackDeath(id, DeathAge)
					world.eventMgr.Send(KILL, id)
				})
			} else if world.currentTick - daemon.LastHeldTokens > daemon.Genome.LifeWithoutFood {
				trackDeath(id, DeathStarvation)
				world.eventMgr.Send(KILL, id)
			}
		}

		if world.numOrganisms > world.maxPop {
			var oldestId string
			var oldestAge int = 0
			for id, daemon := range world.organisms {
				if daemon.Age() > oldestAge {
					oldestAge = daemon.Age()
					oldestId = id
				}
			}
			world.eventMgr.Send(KILL, oldestId)
		}
	})

	for {
		select {
			case <-ctx.Done():
				wg.Wait()
				return
			case event := <-world.reqChannel:
				world.handleRequest(event)
		}
	}
}

func (world *World) TicksPerS() int {
	return world.ticksPerS
}




