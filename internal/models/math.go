package models

// MeanValue calculates the arithmetic mean of a slice of float64 values.
// Returns 0 if the slice is empty
func MeanValue(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
