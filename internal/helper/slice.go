package helper

// MapSlice transforms items in order and returns an empty slice for nil input.
func MapSlice[T, U any](items []T, convert func(T) U) []U {
	result := make([]U, len(items))
	for i, item := range items {
		result[i] = convert(item)
	}
	return result
}
