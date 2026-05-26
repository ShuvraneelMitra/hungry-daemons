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

    AvgCPUHunger float64
    AvgReplicationRate float64
    AvgMutationChance float64
    AvgLifespan float64

    MinCPUHunger int
    MaxCPUHunger int

    DominantLineageID string
    DominantLineageCount int

    AvgFreeCPUTokens float64

    totalFreeTokensObserved int
    tokenObservationCount   int
}

// thankfully no Mutexes in this file since the statistics object will only 
// ever be written by these functions, which will be called ONLY from World!

func CreateStats() *Stats {
	onceStat.Do(func(){
		stats = &Stats{
			DeathsByType: make(map[DeathType]int),
		}
	})
	return stats
}

func (stats *Stats) TrackDeath(reason DeathType) {
	stats.DeathsByType[reason]++
	stats.TotalDeaths++
}

func (stats *Stats) TrackBirth(surname string) {
	stats.TotalBirths++
}

func (stats *Stats) UpdateMetrics() {
	stats.TotalTicks++
}
