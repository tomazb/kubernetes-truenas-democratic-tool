package inventorycache

// cloneSlice returns a shallow copy of a slice so callers cannot mutate cached data.
func cloneSlice[T any](items []T) []T {
	if len(items) == 0 {
		return nil
	}
	return append([]T(nil), items...)
}
