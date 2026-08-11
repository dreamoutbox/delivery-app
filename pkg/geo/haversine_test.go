package geo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateDistanceKm(t *testing.T) {
	// Victory Monument Bangkok (13.7649, 100.5383) to Siam Paragon Bangkok (13.7460, 100.5347)
	// Approx 2.1 km
	dist := CalculateDistanceKm(13.7649, 100.5383, 13.7460, 100.5347)
	assert.InDelta(t, 2.1, dist, 0.3)
}

func TestCalculateDeliveryFeeSatang(t *testing.T) {
	baseFee := 1500 // 15 THB
	perKm := 500    // 5 THB per km

	// 2.1 km distance
	fee := CalculateDeliveryFeeSatang(baseFee, perKm, 13.7649, 100.5383, 13.7460, 100.5347)
	// base 1500 + round(2.1 * 500) = 1500 + 1050 approx = 2550 satang (25.50 THB)
	assert.Greater(t, fee, baseFee)
}
