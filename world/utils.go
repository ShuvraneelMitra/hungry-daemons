package world

import (
	"math/rand/v2"
	"os"
	"github.com/pelletier/go-toml/v2"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type Config struct {
	Env struct {
		InitPop   int `toml:"initial_population"`
		MaxPop    int `toml:"max_population"`
		SimTicks  int `toml:"simulation_ticks"`
		MaxCPU    int `toml:"max_cpu_tokens"`
		LifeExp   int `toml:"life_expectancy"`
		TicksPerS int `toml:"ticks_per_s"`
	} `toml:"env"`

	Genome struct {
		MinCPUHunger int `toml:"min_cpu_hunger"`
		MaxCPUHunger int `toml:"max_cpu_hunger"`

		MinReplicationRate int `toml:"min_replication_rate"`
		MaxReplicationRate int `toml:"max_replication_rate"`

		MinMutationChance float64 `toml:"min_mutation_chance"`
		MaxMutationChance float64 `toml:"max_mutation_chance"`

		MinLifeWithoutFood int `toml:"min_life_without_food"`
		MaxLifeWithoutFood int `toml:"max_life_without_food"`

		MinLifespanRatio float64 `toml:"min_lifespan_ratio"`
		MaxLifespanRatio float64 `toml:"max_lifespan_ratio"`

		MinHoldTime int `toml:"min_hold_time"`
		MaxHoldTime int `toml:"max_hold_time"`
	} `toml:"genome"`

	Simulation struct {
		DeathProb       float64 `toml:"death_prob"`
		ReplicationProb float64 `toml:"replication_prob"`
		CPUReleaseProb  float64 `toml:"cpu_release_prob"`
	} `toml:"simulation"`
}

func ParseConfig(configFile string) Config {
	var cfg Config

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

	return cfg
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

func probablyExecute(probability float64, fn func()) {
	if rand.Float64() < probability {
		fn()
	}
}

func randomScale() float64 {
	if rand.IntN(2) == 0 {
		return 0.5 + rand.Float64()*0.5
	}
	return 1 + rand.Float64()
}

func mutateInt(value int) int {
	if value <= 1 {
		return value
	}

	change := rand.IntN(value)
	return value / 2 + change
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxInt(value, min int) int {
	if value < min {
		return min
	}
	return value
}

func randomIntRange(minVal, maxVal int) int {
	if maxVal <= minVal {
		return minVal
	}

	return rand.IntN(maxVal-minVal+1) + minVal
}

func randomFloatRange(minVal, maxVal float64) float64 {
	if maxVal <= minVal {
		return minVal
	}

	return minVal + rand.Float64()*(maxVal-minVal)
}
