package world

import (
	"context"
	"fmt"
	"maps"
	"math"
	"sync"
	"time"

	. "github.com/ShuvraneelMitra/hungry-daemons/managers"
)

const BUFFER_SZ  = 100

type EventHandler func(world *World, data any)

var (
	once, initializeOnce sync.Once
	handlerTable = map[EventType]EventHandler{
		KILL: func(world *World, data any) {
				info, ok := data.([]any)
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
					info, ok := data.([]any)
					if !ok { return }
					world.Spawn(info[0].(Genome), info[1].(int), world.CurrentTick())
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
	maxCPU         int

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
	ChannelToUI     chan string
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
	maps.Copy(daemons, world.organisms)

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

func NewWorld(cfg Config) (*World, chan string) {
	ctx, cancel := context.WithCancel(context.Background())

	world := &World{
		numOrganisms  : 0,
		numFreeTokens : cfg.Env.MaxCPU,
		lifeExpectancy: cfg.Env.LifeExp,
		ticksPerS     : cfg.Env.TicksPerS,
		currentTick   : 0,
		maxPop        : cfg.Env.MaxPop,
		maxCPU        : cfg.Env.MaxCPU,

		mtx           : &sync.RWMutex{},
		wg            : &sync.WaitGroup{},

		eventMgr      : NewEventManager(),
		organisms     : make(map[string]*Daemon),
		ticker        : time.NewTicker(time.Second / time.Duration(cfg.Env.TicksPerS)),
		reqChannel    : make(chan Event, cfg.Env.MaxPop + 1),
		ctx           : ctx,
		cancelFunc    : cancel,
		shutdownOnce  : sync.Once{},
		done          : make(chan struct{}),

		stats         : CreateStats(),
		ChannelToUI    : make(chan string, BUFFER_SZ),
	}

	// SUBSCRIBE TO EVENTS
	world.eventMgr.Subscribe(KILL, world.reqChannel)
	world.eventMgr.Subscribe(RELEASE_CPU, world.reqChannel)
	world.eventMgr.Subscribe(SPAWN, world.reqChannel)
	world.eventMgr.Subscribe(REQUEST_CPU, world.reqChannel)
	return world, world.ChannelToUI
}

func (world *World) Initialize(cfg Config){
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
				

				minLifespan := int(float64(world.lifeExpectancy) * cfg.Genome.MinLifespanRatio)
				maxLifespan := int(float64(world.lifeExpectancy) * cfg.Genome.MaxLifespanRatio)

				genome := Genome{
					ID                : id,
					Surname           : surname,

					ReplicationRate   : randomIntRange(cfg.Genome.MinReplicationRate, cfg.Genome.MaxReplicationRate),
					CPUHunger         : randomIntRange(cfg.Genome.MinCPUHunger, cfg.Genome.MaxCPUHunger),
					MutationChance    : randomFloatRange(cfg.Genome.MinMutationChance, cfg.Genome.MaxMutationChance),
					LifeWithoutFood   : randomIntRange(cfg.Genome.MinLifeWithoutFood, cfg.Genome.MaxLifeWithoutFood),
					MinimumLifespan   : randomIntRange(minLifespan, maxLifespan),
					MinimumHoldTime   : randomIntRange(cfg.Genome.MinHoldTime, cfg.Genome.MaxHoldTime),

					InstructionSet: nil,
				}

				world.organisms[id] = NewDaemon(genome, world, world.currentTick)

				world.stats.TrackBirth(world.organisms[id].Genome.Surname)	

				if len(world.organisms) == world.numOrganisms {
					break
				}
			}
		})
}

func (world *World) Run(extCtx context.Context, simTicks int) {
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
				
				case <-extCtx.Done():
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

func (world *World) ValidateInvariants() error {
	world.mtx.RLock()
	defer world.mtx.RUnlock()

	if world.numOrganisms != len(world.organisms) {
		return fmt.Errorf(
			"invariant failed: numOrganisms=%d but len(organisms)=%d",
			world.numOrganisms,
			len(world.organisms),
		)
	}

	if world.numOrganisms < 0 {
		return fmt.Errorf("invariant failed: numOrganisms is negative: %d", world.numOrganisms)
	}

	if world.numFreeTokens < 0 {
		return fmt.Errorf("invariant failed: numFreeTokens is negative: %d", world.numFreeTokens)
	}

	if world.numFreeTokens > world.maxCPU {
		return fmt.Errorf(
			"invariant failed: numFreeTokens=%d exceeds max CPU tokens=%d",
			world.numFreeTokens,
			world.maxCPU,
		)
	}

	totalHeldTokens := 0

	for id, daemon := range world.organisms {
		if daemon == nil {
			return fmt.Errorf("invariant failed: daemon %s is nil", id)
		}

		currentTokens, _ := daemon.State()

		if currentTokens < 0 {
			return fmt.Errorf(
				"invariant failed: daemon %s has negative tokens: %d",
				id,
				currentTokens,
			)
		}

		if currentTokens > world.maxCPU {
			return fmt.Errorf(
				"invariant failed: daemon %s has tokens=%d exceeding max CPU=%d",
				id,
				currentTokens,
				world.maxCPU,
			)
		}

		totalHeldTokens += currentTokens
	}

	if world.numFreeTokens + totalHeldTokens > world.maxCPU {
		return fmt.Errorf(
			"invariant failed: free tokens + held tokens = %d, exceeds max CPU tokens=%d",
			world.numFreeTokens+totalHeldTokens,
			world.maxCPU,
		)
	}

	return nil
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
		select {
			case daemon.TickC <- tick:
			default:
				// daemon missed this tick
		}
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

func (world *World) GetPopChan() chan float64 {
	return world.stats.FloatChannel
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
	world.mtx.Unlock()

	tokens, _ := daemon.State()

	world.mtx.Lock()
	world.numFreeTokens += tokens
	world.numOrganisms--
	delete(world.organisms, daemonId)
	world.mtx.Unlock()

	world.stats.TrackDeath(reason, daemon.Age())

	close(daemon.Channel)
	close(daemon.TickC)
	daemon.ClearTokens(world.currentTick)
}

func (world *World) PrintMetrics() {
	stats := world.stats

	var statistics string
	if world.numOrganisms > 0 {
		statistics = fmt.Sprintf(
	`
	----------- FINAL POPULATION METRICS ---------------
	Ticks                    : %d

	Population               : %d

	Total Births             : %d
	Total Deaths             : %d

	Deaths By Age            : %d
	Deaths By Starvation     : %d

	Average CPU Hunger       : %.4f
	Average Replication Rate : %.4f
	Average Mutation Chance  : %.4f
	Average MinLifespan      : %.4f
	Average Current Age      : %.4f
	Average Death Age        : %.4f

	Minimum CPU Hunger       : %d
	Maximum CPU Hunger       : %d

	Dominant Lineage ID      : %s
	Dominant Lineage Count   : %d

	Average Free CPU Tokens  : %.4f

	=======================================================
	`,
			stats.TotalTicks,

			world.numOrganisms,

			stats.TotalBirths,
			stats.TotalDeaths,

			stats.DeathsByType[DeathAge],
			stats.DeathsByType[DeathStarvation],
			
			stats.AvgCPUHunger,
			stats.AvgReplicationRate,
			stats.AvgMutationChance,
			stats.AvgConfiguredLifespan,
			stats.AvgAge,
			stats.AvgDeathAge,

			stats.MinCPUHunger,
			stats.MaxCPUHunger,

			stats.DominantLineageID,
			stats.DominantLineageCount,

			stats.AvgFreeCPUTokens,
		)
	} else {
		statistics = fmt.Sprintf(
		`
		----------- FINAL POPULATION METRICS ---------------
		Ticks                    : %d

		Population               : %d

		Total Births             : %d
		Total Deaths             : %d

		Deaths By Age            : %d
		Deaths By Starvation     : %d

		Average Death Age        : %.4f
		Average Free CPU Tokens  : %.4f

		=======================================================
		`,
				stats.TotalTicks,

				world.numOrganisms,

				stats.TotalBirths,
				stats.TotalDeaths,

				stats.DeathsByType[DeathAge],
				stats.DeathsByType[DeathStarvation],
				
				stats.AvgDeathAge,
				stats.AvgFreeCPUTokens,
			)
		}
	world.ChannelToUI<-statistics
}

func (world *World) ReleaseCpu(daemonId string) {
	world.mtx.Lock()

	daemon, ok := world.organisms[daemonId]
	if !ok {
		world.mtx.Unlock()
		return
	}
	world.mtx.Unlock()

	tokens, _ := daemon.State()

	world.mtx.Lock()
	world.numFreeTokens += tokens
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

		world.PrintMetrics()
	})
}

func (world *World) Spawn(genome Genome, generation int, tick int) {
	world.mtx.Lock()

	if world.numOrganisms >= world.maxPop {
		world.mtx.Unlock()
		return 
	}

	daemon := NewDaemon(genome, world, tick)
	daemon.Generation = generation

	if _, exists := world.organisms[genome.ID]; exists {
		world.mtx.Unlock()
		return
	}

	world.organisms[daemon.Genome.ID] = daemon
	
	world.numOrganisms++
	world.mtx.Unlock()

	world.stats.TrackBirth(genome.Surname)
	world.wg.Go(func() { daemon.Run(world.ctx) })
}

func (world *World) Tick(ctx context.Context) {
	world.mtx.Lock()
	world.currentTick++

	world.ChannelToUI<-fmt.Sprintf("Tick number %d", world.currentTick)
	world.ChannelToUI<-fmt.Sprintf("Population %d\n", world.numOrganisms)

	currentTick := world.currentTick
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

		// if population > world.maxPop {
		// 	var oldestId string
		// 	var oldestAge int = 0
		// 	for _, daemon := range organisms {
		// 		if daemon.age > oldestAge {
		// 			oldestAge = daemon.age
		// 			oldestId = daemon.id
		// 		}
		// 	}
		// 	world.eventMgr.Send(KILL, []any{
		// 		oldestId, DeathCulling,
		// 	})
		// } //This is actually a useless snippet since Spawn won't work if pop > maxpop
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
		world.mtx.RLock()
		if world.numOrganisms == 0 {
			world.ChannelToUI<-"Population died out! World ends here\n"
			world.mtx.RUnlock()
			world.Shutdown()
		} else {
			world.mtx.RUnlock()
		}
		world.UpdateMetrics()

		if err := world.ValidateInvariants(); err != nil {
			panic(err)
		}
		wg.Wait()
}

func (world *World) TicksPerS() int {
	return world.ticksPerS
}

func (world *World) UpdateMetrics() {
	world.mtx.RLock()
	defer world.mtx.RUnlock()

	population := len(world.organisms)

	world.stats.AdvanceTick()
	world.stats.ObserveFreeTokens(world.numFreeTokens)

	if population == 0 {
		world.stats.SetGenomeAverages(0, 0, 0, 0, 0)
		world.stats.SetCPUHungerRange(0, 0)
		world.stats.SetDominantLineage("", 0)
		return
	}

	totalCPUHunger := 0
	totalReplicationRate := 0
	totalMutationChance := 0.0
	totalLifespan := 0
	totalAge := 0

	minCPUHunger := math.MaxInt
	maxCPUHunger := math.MinInt

	lineageCounts := make(map[string]int)

	for _, daemon := range world.organisms {
		genome := daemon.Genome

		totalCPUHunger += genome.CPUHunger
		totalReplicationRate += genome.ReplicationRate
		totalMutationChance += genome.MutationChance
		totalLifespan += genome.MinimumLifespan

		if genome.CPUHunger < minCPUHunger {
			minCPUHunger = genome.CPUHunger
		}

		if genome.CPUHunger > maxCPUHunger {
			maxCPUHunger = genome.CPUHunger
		}

		lineageCounts[genome.Surname]++
		totalAge += daemon.Age()
	}

	dominantLineage := ""
	dominantCount := 0

	for lineage, count := range lineageCounts {
		if count > dominantCount {
			dominantLineage = lineage
			dominantCount = count
		}
	}

	world.stats.SetGenomeAverages(
		float64(totalCPUHunger)/float64(population),
		float64(totalReplicationRate)/float64(population),
		totalMutationChance/float64(population),
		float64(totalLifespan)/float64(population),
		float64(totalAge) / float64(population),
	)

	world.stats.SetCPUHungerRange(minCPUHunger, maxCPUHunger)
	world.stats.SetDominantLineage(dominantLineage, dominantCount)

	world.stats.FloatChannel<-float64(population)
}



