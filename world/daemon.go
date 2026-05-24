package world

import (

)

type Instruction struct {
	Operation string
	Operand1 string
	Operand2 string
	Operand3 string
}

type Genome struct {
    ID              string
    ParentID        string
    ReplicationRate float64
    CPUHunger       int
    MutationChance  float64
    MinimumLifespan int
	InstructionSet  []Instruction
}

type Daemon struct {
    Genome          Genome
    CurrentTokens   int
    CreatedTick     int // essentially Birthday!
	Env             Environment
}

func (daemon *Daemon) Age() int {
	return daemon.Env.CurrentTick() - daemon.CreatedTick
}
