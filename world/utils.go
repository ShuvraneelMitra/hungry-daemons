package world

import (
	"math/rand/v2"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

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

	change := rand.IntN(value) - value/2
	return value + change
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
