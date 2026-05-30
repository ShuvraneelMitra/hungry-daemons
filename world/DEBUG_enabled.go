//go:build DEBUG

package world

import "fmt"

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

