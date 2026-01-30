package maps

func MapValues[K comparable, From, To any](m map[K]From, valueMapper func(From) To) (result map[K]To) {
	result = make(map[K]To, len(m))
	for k, v := range m {
		result[k] = valueMapper(v)
	}
	return
}
