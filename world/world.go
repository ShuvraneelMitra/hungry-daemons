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

const (
	BUFFER_SZ  = 1000
	LOG_ONCE_IN_X_TICKS = 15
)

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
	DeathProb() float64
	ReplicationProb() float64
	CPUReleaseProb() float64
}

type World struct {
	numOrganisms   int
	numFreeTokens  int
	lifeExpectancy int
	ticksPerS      int
	currentTick    int
	maxPop         int
	maxCPU         int
	
	deathProb       float64 // Probability that at the current ticker if age > lifeExpectancy the organism dies
	replicationProb float64
	cpuReleaseProb  float64

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
	ChannelsToUI   map[string]chan any
	Census         map[string]*LineageStats
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

func NewWorld(cfg Config) (*World, map[string]chan any) {
	ctx, cancel := context.WithCancel(context.Background())

	world := &World{
		numOrganisms    : 0,
		numFreeTokens   : cfg.Env.MaxCPU,
		lifeExpectancy  : cfg.Env.LifeExp,
		ticksPerS       : cfg.Env.TicksPerS,
		currentTick     : 0,
		maxPop          : cfg.Env.MaxPop,
		maxCPU          : cfg.Env.MaxCPU,

		deathProb       : cfg.Simulation.DeathProb,
        replicationProb : cfg.Simulation.ReplicationProb,
        cpuReleaseProb  : cfg.Simulation.CPUReleaseProb,

		mtx             : &sync.RWMutex{},
		wg              : &sync.WaitGroup{},

		eventMgr        : NewEventManager(),
		organisms       : make(map[string]*Daemon),
		ticker          : time.NewTicker(time.Second / time.Duration(cfg.Env.TicksPerS)),
		reqChannel      : make(chan Event, cfg.Env.MaxPop + 1),
		ctx             : ctx,
		cancelFunc      : cancel,
		shutdownOnce    : sync.Once{},
		done            : make(chan struct{}),

		stats           : CreateStats(),
		ChannelsToUI    : map[string]chan any{
									"metrics": make(chan any, BUFFER_SZ),
									"logs": make(chan any, BUFFER_SZ),
									"topK": make(chan any, BUFFER_SZ),
							},
		Census			: make(map[string]*LineageStats),
	}

	world.ChannelsToUI["populationData"] = world.stats.PopulationDataChan

	// SUBSCRIBE TO EVENTS
	world.eventMgr.Subscribe(KILL, world.reqChannel)
	world.eventMgr.Subscribe(RELEASE_CPU, world.reqChannel)
	world.eventMgr.Subscribe(SPAWN, world.reqChannel)
	world.eventMgr.Subscribe(REQUEST_CPU, world.reqChannel)
	return world, world.ChannelsToUI
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
				world.Census[genome.Surname] = &LineageStats{}
				world.Census[genome.Surname].TrackBirth(genome, 0)

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

func (world *World) CPUReleaseProb() float64 {
	return world.cpuReleaseProb
}

func (world *World) CurrentTick() int {
	world.mtx.RLock()
	defer world.mtx.RUnlock()
	return world.currentTick
}

func (world *World) DeathProb() float64 {
	return world.deathProb
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

func (world *World) GetPopChan() chan any {
	return world.stats.PopulationDataChan
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

	world.stats.TrackDeath(daemon.Genome.Surname, reason, daemon.Age())
	if lineage := world.Census[daemon.Genome.Surname]; lineage != nil {
		lineage.TrackDeath(daemon.Age())
	}

	close(daemon.Channel)
	close(daemon.TickC)
	daemon.ClearTokens(world.currentTick)
}

func (world *World) PrintLineageMetrics(topN int) {
	lineages := world.stats.LineageTracker.TopK(topN, true)
	if len(lineages) == 0 {
		return
	}

	world.ChannelsToUI["metrics"]<-"	----------- LINEAGE METRICS ---------------"
	for i := 0; i < min(topN, len(lineages)); i++  {
		l := lineages[i].Surname
		info := world.Census[l]

		world.ChannelsToUI["metrics"]<-fmt.Sprintf(
		`
	Lineage                 : %s

	Population              : %d

	Total Births            : %d
	Total Deaths            : %d

	Living Avg CPU Hunger   : %.3f
	Living Avg Repl Rate    : %.3f
	Living Avg Mutation     : %.3f

	Historical Avg CPU      : %.3f
	Historical Avg Repl     : %.3f
	Historical Avg Mutation : %.3f

	Average Death Age       : %.3f

	Max Generation          : %d

	--------------------------------------------------
		`,
		info.LineageID,

		info.Population,

		info.TotalBirths,
		info.TotalDeaths,

		info.LivingAvgCPUHunger,
		info.LivingAvgReplicationRate,
		info.LivingAvgMutationChance,

		info.HistoricalAvgCPUHunger,
		info.HistoricalAvgReplicationRate,
		info.HistoricalAvgMutationChance,

		info.AverageDeathAge,

		info.MaxGeneration,
		)
	}
}

func (world *World) PrintMetrics() {
	stats := world.stats
	dominant := stats.LineageTracker.TopK(1, true)[0]

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

			dominant.Surname,
			dominant.Count,

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
			
			world.ChannelsToUI["topK"]<-world.stats.LineageTracker.TopK(5, true)
		}
	world.ChannelsToUI["metrics"]<-statistics
	world.ChannelsToUI["metrics"]<-"\n\n"
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

func (world *World) ReplicationProb() float64 {
	return world.replicationProb
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
		world.PrintLineageMetrics(5)
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
	if _, exists := world.Census[genome.Surname]; !exists {
		world.Census[genome.Surname] = &LineageStats{}
	}
	world.Census[genome.Surname].TrackBirth(daemon.Genome, daemon.Generation)
	world.wg.Go(func() { daemon.Run(world.ctx) })
}

func (world *World) Tick(ctx context.Context) {
	world.mtx.Lock()
	world.currentTick++

	domLineage := world.stats.LineageTracker.TopK(1, true)[0]

	if world.currentTick % LOG_ONCE_IN_X_TICKS == 0 {
		world.ChannelsToUI["logs"]<-fmt.Sprintf("Tick number %d", world.currentTick)
		world.ChannelsToUI["logs"]<-fmt.Sprintf("Population %d", world.numOrganisms)
		world.ChannelsToUI["logs"]<-fmt.Sprintf("Dominant Lineage %s, Count %d\n", 
											domLineage.Surname, domLineage.Count)
	}

	currentTick := world.currentTick
	world.mtx.Unlock()

	world.broadcastTick(currentTick)


	var wg sync.WaitGroup
	wg.Go(func() { // Killing in the name of
		organisms := world.snapshotOrganisms()

		for _, daemon := range organisms {
			if daemon.age > daemon.minimumLifespan {
				probablyExecute(world.deathProb, func(){
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
			world.ChannelsToUI["logs"]<-"Population died out! World ends here\n"
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

func (world *World) UpdateCurPopAverages() {
	world.mtx.RLock()
	defer world.mtx.RUnlock()

	for _, lineageStat := range world.Census {
		lineageStat.Population = 0
		lineageStat.totalLivingCPUHunger = 0
		lineageStat.totalLivingReplicationRate = 0
		lineageStat.totalLivingMutationChance = 0

		lineageStat.LivingAvgCPUHunger = 0
		lineageStat.LivingAvgReplicationRate = 0
		lineageStat.LivingAvgMutationChance = 0
	}

	for _, daemon := range world.organisms {
		lineageID := daemon.Genome.Surname
		lineage := world.Census[lineageID]

		lineage.Population++

		lineage.totalLivingCPUHunger += daemon.Genome.CPUHunger
		lineage.totalLivingReplicationRate += daemon.Genome.ReplicationRate
		lineage.totalLivingMutationChance += daemon.Genome.MutationChance
	}

	for _, lineage := range world.Census {
		if lineage.Population == 0 {
			continue
		}

		lineage.LivingAvgCPUHunger = float64(lineage.totalLivingCPUHunger) / float64(lineage.Population)
		lineage.LivingAvgReplicationRate = float64(lineage.totalLivingReplicationRate) / float64(lineage.Population)
		lineage.LivingAvgMutationChance = lineage.totalLivingMutationChance / float64(lineage.Population)
	}
}


func (world *World) UpdateMetrics() {
	world.mtx.RLock()

	population := len(world.organisms)

	world.stats.AdvanceTick()
	world.stats.ObserveFreeTokens(world.numFreeTokens)

	if population == 0 {
		world.stats.SetGenomeAverages(0, 0, 0, 0, 0)
		world.stats.SetCPUHungerRange(0, 0)
		world.mtx.RUnlock()
		return
	}

	totalCPUHunger := 0
	totalReplicationRate := 0
	totalMutationChance := 0.0
	totalLifespan := 0
	totalAge := 0

	minCPUHunger := math.MaxInt
	maxCPUHunger := math.MinInt

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
		totalAge += daemon.Age()
	}

	world.stats.SetGenomeAverages(
		float64(totalCPUHunger)/float64(population),
		float64(totalReplicationRate)/float64(population),
		totalMutationChance/float64(population),
		float64(totalLifespan)/float64(population),
		float64(totalAge) / float64(population),
	)

	world.stats.SetCPUHungerRange(minCPUHunger, maxCPUHunger)
	world.mtx.RUnlock()

	world.UpdateCurPopAverages()

	world.ChannelsToUI["populationData"]<-float64(population)
	world.ChannelsToUI["topK"]<-world.stats.LineageTracker.TopK(5, true)
}



