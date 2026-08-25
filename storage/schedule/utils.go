package schedule

func repeatWith[T any](n int, f func() T) []T {
	result := make([]T, n)
	for i := range result {
		result[i] = f()
	}
	return result
}
