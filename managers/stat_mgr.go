package managers

import (
	"sync"
)

type DeathType int
// Death Reasons
const (
    DeathAge DeathType = iota
    DeathStarvation
    DeathCulling
)

var (
	onceStat sync.Once
	stats *Stats
)

type Stats struct {
	TotalTicks int

	TotalBirths int
	TotalDeaths int

	DeathsByType map[DeathType]int

	AvgCPUHunger           float64
	AvgReplicationRate     float64
	AvgMutationChance      float64
	AvgConfiguredLifespan  float64
    AvgAge                 float64
    AvgDeathAge            float64

    totalDeathAge          int

	MinCPUHunger           int
	MaxCPUHunger           int

	AvgFreeCPUTokens       float64

	totalFreeTokensObserved int
	tokenObservationCount   int

	PopulationDataChan      chan any

	LineageTracker          *SortedMap
}

// thankfully no Mutexes in this file since the statistics object will only 
// ever be written by these functions, which will be called ONLY from World!

func CreateStats() *Stats {
	onceStat.Do(func(){
		stats = &Stats{
			DeathsByType: make(map[DeathType]int),
			PopulationDataChan: make(chan any),
			LineageTracker: &SortedMap{
				data : make(map[string]int),
				lineages: make([]LineageCount, 0),
			},
		}
	})
	return stats
}

func (stats *Stats) AdvanceTick() {
	stats.TotalTicks++
}

func (stats *Stats) ObserveFreeTokens(freeTokens int) {
	stats.totalFreeTokensObserved += freeTokens
	stats.tokenObservationCount++

	stats.AvgFreeCPUTokens =
		float64(stats.totalFreeTokensObserved) /
			float64(stats.tokenObservationCount)
}

func (stats *Stats) SetCPUHungerRange(minCPUHunger, maxCPUHunger int) {
	stats.MinCPUHunger = minCPUHunger
	stats.MaxCPUHunger = maxCPUHunger
}

func (stats *Stats) SetGenomeAverages(
	avgCPUHunger float64,
	avgReplicationRate float64,
	avgMutationChance float64,
	avgLifespan float64,
    avgAge float64,
) {
	stats.AvgCPUHunger = avgCPUHunger
	stats.AvgReplicationRate = avgReplicationRate
	stats.AvgMutationChance = avgMutationChance
	stats.AvgConfiguredLifespan = avgLifespan
    stats.AvgAge = avgAge
}

func (stats *Stats) TrackBirth(surname string) {
	stats.TotalBirths++
	pop, ok := stats.LineageTracker.Get(surname)
	if !ok {
		stats.LineageTracker.Set(surname, 1)
	} else{
		stats.LineageTracker.Set(surname, pop + 1)
	}
}

func (stats *Stats) TrackDeath(surname string, reason DeathType, age int) {
    if age == 0 {
        stats.DeathsByType[reason]++
	    stats.TotalDeaths++
        return
    }

	stats.DeathsByType[reason]++
	stats.TotalDeaths++

	stats.totalDeathAge += age

	stats.AvgDeathAge =
		float64(stats.totalDeathAge) /
		float64(stats.TotalDeaths)

	pop, ok := stats.LineageTracker.Get(surname)
	if !ok {
		return
	} else{
		if pop == 1 {
			stats.LineageTracker.Delete(surname)
		} else {
			stats.LineageTracker.Set(surname, pop - 1) 
		}
	}
}

