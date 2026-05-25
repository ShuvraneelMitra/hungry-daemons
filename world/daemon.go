package world

import (
    . "github.com/ShuvraneelMitra/hungry-daemons/managers"
    "context"
    "time"
    "math/rand/v2"
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
    ReplicationRate float64
    CPUHunger       int
    MutationChance  float64
    MinimumLifespan int
    LifeWithoutFood int // max number of ticks it can survive without food (cputokens)
	InstructionSet  []Instruction
}

type Daemon struct {
    Genome             Genome
    CurrentTokens      int
    CreatedTick        int // essentially Birthday!
    LastHeldTokens     int // tick

	Env                Environment
    Channel            chan Event
}

func NewDaemon(genome Genome, world Environment, tick int) *Daemon {
    daemon := &Daemon{ 
        Genome        : genome,
        CurrentTokens : 0,
        CreatedTick   : tick,
        LastHeldTokens: -1,

        Env           : world,
        Channel       : make(chan Event),
    }

    return daemon
}

func (daemon *Daemon) Age() int {
	return daemon.Env.CurrentTick() - daemon.CreatedTick
}

func (daemon *Daemon) MutateGenome(parent Genome) Genome {
	child := parent

	probablyExecute(parent.MutationChance, func() {
		child.ReplicationRate *= randomScale()
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


func (daemon *Daemon) Run(ctx context.Context, ticker *time.Ticker) {
    for {
            select {
                case <-ctx.Done():
                    return

                case <-ticker.C:
					tickCtx, cancel := context.WithTimeout(ctx, time.Second/time.Duration(daemon.Env.TicksPerS()))
					
            }
		}
}

func (daemon *Daemon) Replicate() {
    childGenome := daemon.MutateGenome(daemon.Genome)
    daemon.Env.SendSignal(SPAWN, childGenome)
}
