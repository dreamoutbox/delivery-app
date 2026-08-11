package geo

import (
	"math"
)

const earthRadiusKm = 6371.0088

// CalculateDistanceKm returns the distance in kilometers between two lat/lng points using the Haversine formula.
func CalculateDistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0

	radLat1 := lat1 * math.Pi / 180.0
	radLat2 := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(radLat1)*math.Cos(radLat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// CalculateDeliveryFeeSatang calculates total delivery fee in satang given base fee, rate per km, and coordinates.
func CalculateDeliveryFeeSatang(baseFeeSatang, perKmSatang int, lat1, lng1, lat2, lng2 float64) int {
	distKm := CalculateDistanceKm(lat1, lng1, lat2, lng2)
	distanceFee := math.Round(distKm * float64(perKmSatang))
	return baseFeeSatang + int(distanceFee)
}
