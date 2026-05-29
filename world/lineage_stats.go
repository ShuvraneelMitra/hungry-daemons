package world

// ------------ LINEAGE STATS --------------------
type LineageStats struct {
	LineageID string

	Population int

	TotalBirths int
	TotalDeaths int

	LivingAvgCPUHunger       float64
	LivingAvgReplicationRate float64
	LivingAvgMutationChance  float64

	HistoricalAvgCPUHunger       float64
	HistoricalAvgReplicationRate float64
	HistoricalAvgMutationChance  float64

	AverageDeathAge float64

	MaxGeneration int

	totalLivingCPUHunger       int
	totalLivingReplicationRate int
	totalLivingMutationChance  float64

	totalHistoricalCPUHunger       int
	totalHistoricalReplicationRate int
	totalHistoricalMutationChance  float64

	totalDeathAge int
}

func (ls *LineageStats) TrackBirth(genome Genome, gen int) {
	ls.LineageID = genome.Surname
	ls.TotalBirths++

	ls.totalHistoricalCPUHunger += genome.CPUHunger
	ls.totalHistoricalReplicationRate += genome.ReplicationRate
	ls.totalHistoricalMutationChance += genome.MutationChance

	ls.HistoricalAvgCPUHunger =
		float64(ls.totalHistoricalCPUHunger) / float64(ls.TotalBirths)

	ls.HistoricalAvgReplicationRate =
		float64(ls.totalHistoricalReplicationRate) / float64(ls.TotalBirths)

	ls.HistoricalAvgMutationChance =
		ls.totalHistoricalMutationChance / float64(ls.TotalBirths)

	if gen > ls.MaxGeneration {
		ls.MaxGeneration = gen
	}
}

func (ls *LineageStats) TrackDeath(age int) {
	ls.TotalDeaths++
	ls.totalDeathAge += age

	ls.AverageDeathAge =
		float64(ls.totalDeathAge) / float64(ls.TotalDeaths)
}

