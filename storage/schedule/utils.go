package schedule

type number interface {
	~int | ~int64 | ~float64
}

func sumBy[T any, N number](s []T, f func(T) N) N {
	var total N
	for _, v := range s {
		total += f(v)
	}
	return total
}

func repeatWith[T any](n int, f func() T) []T {
	result := make([]T, n)
	for i := range result {
		result[i] = f()
	}
	return result
}
