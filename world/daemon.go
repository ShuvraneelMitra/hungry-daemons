package world

import (
    . "github.com/ShuvraneelMitra/hungry-daemons/managers"
    "context"
    "time"
    "sync"
)

const (
    CPU_RELEASE_PROB = 0.9
	REPLICATION_PROB = 0.9
)

type Instruction struct {
	Operation string
	Operand1 string
	Operand2 string
	Operand3 string
}

type Genome struct {
    ID              string
    Surname         string
    ReplicationRate int
    CPUHunger       int
    MutationChance  float64
    MinimumHoldTime int // how long it holds on to CPU tokens before releasing
    MinimumLifespan int
    LifeWithoutFood int // max number of ticks it can survive without food (cputokens)
	InstructionSet  []Instruction
}

type Daemon struct {
    Genome             Genome
	Generation         int
    CurrentTokens      int
    CreatedTick        int // essentially Birthday!
    LastHeldTokens     int // tick

    mtx                *sync.RWMutex

	Env                Environment
    Channel            chan Event
    TickC              chan int
}

func NewDaemon(genome Genome, world Environment, tick int) *Daemon {
    daemon := &Daemon{ 
        Genome        : genome,
		Generation    : 0,
        CurrentTokens : 0,
        CreatedTick   : tick,
        LastHeldTokens: tick,

		mtx           : &sync.RWMutex{},

        Env           : world,
        Channel       : make(chan Event),
        TickC         : make(chan int, 1),
    }

    return daemon
}

func (daemon *Daemon) Age() int {
	return daemon.Env.CurrentTick() - daemon.CreatedTick
}

func (daemon *Daemon) MutateGenome(parent Genome) Genome {
	child := parent

	probablyExecute(parent.MutationChance, func() {
		child.ReplicationRate *= mutateInt(parent.ReplicationRate)
		child.MutationChance *= randomScale()

		child.CPUHunger = mutateInt(parent.CPUHunger)
		child.MinimumLifespan = mutateInt(parent.MinimumLifespan)
		child.LifeWithoutFood = mutateInt(parent.LifeWithoutFood)

		child.MutationChance = clampFloat(child.MutationChance, 0, 1)
		child.CPUHunger = maxInt(child.CPUHunger, 1)
		child.MinimumLifespan = maxInt(child.MinimumLifespan, 1)
		child.LifeWithoutFood = maxInt(child.LifeWithoutFood, 1)
	})

	return child
}


func (daemon *Daemon) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case tick, ok := <-daemon.TickC:
            if !ok { return }
            tickCtx, cancel := context.WithTimeout(ctx, time.Second / time.Duration(daemon.Env.TicksPerS()))
			daemon.Tick(tickCtx, tick)
            cancel()
		}
	}
}

func (daemon *Daemon) Replicate() {
    newGenome := daemon.Genome
    newGenome.ID = generateRandomString(len(daemon.Genome.ID))

    childGenome := daemon.MutateGenome(newGenome)
    daemon.Env.SendSignal(SPAWN, []any{
		childGenome, daemon.Generation + 1,
	})
}

func (daemon *Daemon) State() (tokens int, lastHeld int) {
	daemon.mtx.RLock()
	defer daemon.mtx.RUnlock()

	return daemon.CurrentTokens, daemon.LastHeldTokens
}

func (daemon *Daemon) SetTokens(tokens int, tick int) {
	daemon.mtx.Lock()
	defer daemon.mtx.Unlock()

	daemon.CurrentTokens = tokens
	if tokens > 0 {
		daemon.LastHeldTokens = tick
	}
}

func (daemon *Daemon) ClearTokens(tick int) {
	daemon.mtx.Lock()
	defer daemon.mtx.Unlock()

	daemon.CurrentTokens = 0
    daemon.LastHeldTokens = tick
}


func (daemon *Daemon) Tick(ctx context.Context, tick int) {
	select {
	case <-ctx.Done():
		return
	default:
	}

    currentTokens, lastHeldTokens := daemon.State()

	if currentTokens == 0 {
		daemon.Env.SendSignal(REQUEST_CPU, []any{
			daemon.Genome.ID,
			daemon.Genome.CPUHunger,
		})
		return
	}

	if tick - lastHeldTokens >= daemon.Genome.MinimumHoldTime {
		probablyExecute(REPLICATION_PROB, func() {
			for range daemon.Genome.ReplicationRate {
				daemon.Replicate()
			}
		})

		probablyExecute(CPU_RELEASE_PROB, func() {
			daemon.Env.SendSignal(RELEASE_CPU, daemon.Genome.ID)
		})
		return
	}

	daemon.mtx.Lock()
    daemon.LastHeldTokens = tick
    daemon.mtx.Unlock()
}
