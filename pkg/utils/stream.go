package utils

type Stream[T any] struct {
	Array []T
}

type MapStream[K comparable, V any] struct {
	Values map[K]V
}

func NewStream[T any](values []T) *Stream[T] {
	return &Stream[T]{
		Array: values,
	}
}

func NewMapStream[K comparable, V any](values map[K]V) *MapStream[K, V] {
	return &MapStream[K, V]{
		Values: values,
	}
}

func (s *Stream[T]) Filter(condition func(T) bool) *Stream[T] {
	filteredValues := make([]T, 0, len(s.Array))
	for _, v := range s.Array {
		if condition(v) {
			filteredValues = append(filteredValues, v)
		}
	}
	return &Stream[T]{
		Array: filteredValues,
	}
}

func (s *MapStream[K, V]) Filter(condition func(K, V) bool) *MapStream[K, V] {
	filteredValues := make(map[K]V)
	for k, v := range s.Values {
		if condition(k, v) {
			filteredValues[k] = v
		}
	}
	return &MapStream[K, V]{
		Values: filteredValues,
	}
}

func (s *MapStream[K, V]) GetValues() *Stream[V] {
	var values []V
	for _, v := range s.Values {
		values = append(values, v)
	}
	return &Stream[V]{
		Array: values,
	}
}

func (s *MapStream[K, V]) GetKeys() *Stream[K] {
	var keys []K
	for k, _ := range s.Values {
		keys = append(keys, k)
	}
	return &Stream[K]{
		Array: keys,
	}
}

func (s *Stream[T]) ForEach(f func(T)) {
	for _, v := range s.Array {
		f(v)
	}
}

func (s *MapStream[K, V]) ForEach(f func(K, V)) {
	for k, v := range s.Values {
		f(k, v)
	}
}

func (s *Stream[T]) All(condition func(T) bool) bool {
	for _, v := range s.Array {
		if !condition(v) {
			return false
		}
	}
	return true
}

func (s *Stream[T]) Map(f func(T) T) *Stream[T] {
	var mappedValues []T
	for _, v := range s.Array {
		mappedValues = append(mappedValues, f(v))
	}
	return &Stream[T]{
		Array: mappedValues,
	}
}

func (s *Stream[T]) MapToInt(f func(T) int) *Stream[int] {
	var mappedValues []int
	for _, v := range s.Array {
		mappedValues = append(mappedValues, f(v))
	}
	return &Stream[int]{
		Array: mappedValues,
	}
}

func (s *Stream[T]) MapToFloat64(f func(T) float64) *Stream[float64] {
	var mappedValues []float64
	for _, v := range s.Array {
		mappedValues = append(mappedValues, f(v))
	}
	return &Stream[float64]{
		Array: mappedValues,
	}
}

func (s *Stream[T]) MapToBool(f func(T) bool) *Stream[bool] {
	var mappedValues []bool
	for _, v := range s.Array {
		mappedValues = append(mappedValues, f(v))
	}
	return &Stream[bool]{
		Array: mappedValues,
	}
}

func (s *Stream[T]) MapToString(f func(T) string) *Stream[string] {
	var mappedValues []string
	for _, v := range s.Array {
		mappedValues = append(mappedValues, f(v))
	}
	return &Stream[string]{
		Array: mappedValues,
	}
}

func (s *Stream[T]) MapToList(f func(T) []T) *Stream[T] {
	var mappedValues []T
	for _, v := range s.Array {
		mappedValues = append(mappedValues, f(v)...)
	}
	return &Stream[T]{
		Array: mappedValues,
	}
}

func (s *Stream[T]) Reduce(f func(T, T) T) *T {
	if len(s.Array) == 0 {
		return nil
	}
	result := s.Array[0]
	for i := 1; i < len(s.Array); i++ {
		result = f(result, s.Array[i])
	}
	return &result
}

func FilterMap[K comparable, V any](m map[K]V, condition func(K, V) bool) map[K]V {
	filteredMap := make(map[K]V)
	for k, v := range m {
		if condition(k, v) {
			filteredMap[k] = v
		}
	}
	return filteredMap
}

func GetValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func GetKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func Map[T any, O any](items []T, f func(T) O) []O {
	result := make([]O, len(items))
	for i, item := range items {
		result[i] = f(item)
	}
	return result
}
