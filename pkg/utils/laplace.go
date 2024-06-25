package utils

import (
	"math"
	"math/rand"
	"time"
)

// Initialize the random number generator
func init() {
	rand.Seed(time.Now().UnixNano())
}

// Function to generate Laplace noise
func laplaceMechanism(value, sensitivity, epsilon float64) float64 {
	scale := sensitivity / epsilon
	u := rand.Float64() - 0.5
	return value - scale*math.Copysign(math.Log(1-2*math.Abs(u)), u)
}

// LaplaceNoise adds Laplace noise to a given value
func LaplaceNoise(value, epsilon, delta float64) float64 {
	b := epsilon / math.Sqrt(2*math.Log(1.25/delta))
	u := rand.Float64() - 0.5
	sign := 1.0
	if u < 0 {
		sign = -1.0
	}
	return value - b*sign*math.Log(1-2*math.Abs(u))
}
