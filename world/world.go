package world

import (
	"math/rand/v2"
	"os"
	"sync"
	"time"

	. "github.com/ShuvraneelMitra/hungry-daemons/managers"
	"github.com/pelletier/go-toml/v2"
)

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
)

const ID_LENGTH = 10

type Environment interface {
	CurrentTick() int
	SendSignal(id int, signal EventType) 
}

type World struct {
	numOrganisms   int
	numFreeTokens  int
	lifeExpectancy int
	ticksPerS      int
	currentTick    int

    organisms      map[string]*Daemon
	mtx            sync.RWMutex
	eventMgr       *EventManager
	ticker         *time.Ticker
}

func NewWorld(configFile string) *World {
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

	world := &World{
		numOrganisms  : 0,
		numFreeTokens : cfg.Env.MaxCPU,
		lifeExpectancy: cfg.Env.LifeExp,
		ticksPerS     : cfg.Env.TicksPerS,
		currentTick   : 0,
		eventMgr      : NewEventManager(),
		organisms     : make(map[string]*Daemon),
		ticker        : time.NewTicker(time.Second / time.Duration(cfg.Env.TicksPerS)),
	}

	return world
}

func (world *World) Initialize(tick int){
	initializeOnce.Do(
		func(){	
			world.numOrganisms = cfg.Env.InitPop
			/*
			Why do this here and not in NewWorld()? Suppose you call 
			NewWorld() but forget to call Initialize(). Then the World
			object will have numOrganisms > 0 but an empty organisms map.
			*/
			
			for {
				id := generateRandomString(ID_LENGTH)
				if _, exists := world.organisms[id]; exists {
					continue
				}
				
				genome := Genome{
					ID               : id,
					ParentID         : "",
					ReplicationRate  : rand.Float64(),
					CPUHunger        : rand.IntN(cfg.Env.MaxCPU),
					MutationChance   : rand.Float64(),
					MinimumLifespan  : rand.IntN(world.lifeExpectancy),
					InstructionSet   : nil,
				}
				world.organisms[id] = &Daemon{
					Genome       : genome,
					CurrentTokens: 0,
					CreatedTick  : tick,
				}

				if len(world.organisms) == world.numOrganisms {
					break
				}
			}
		})
}

func (world *World) AllocateTokens(daemonId string, tokens int) int {
	if world.numFreeTokens < tokens || tokens <= 0 {
		return -1
	}

	world.mtx.Lock()
	defer world.mtx.Unlock()
	_, ok := world.organisms[daemonId]
	if !ok {
		return -1
	}

	world.numFreeTokens -= tokens
	world.organisms[daemonId].CurrentTokens += tokens
	return 0
}

func (world *World) Kill(daemonId string) {
	world.eventMgr.Send(KILL, daemonId)
}

func (world *World) CurrentTick() int {
	world.mtx.RLock()
	defer world.mtx.RUnlock()
	return world.currentTick
}

func (world *World) SendSignal(daemonId int, signal EventType) {
	world.mtx.Lock()
	defer world.mtx.Unlock()

	world.eventMgr.Send(signal, daemonId)
}




