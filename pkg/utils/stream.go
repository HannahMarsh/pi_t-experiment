package utils

import (
	"cmp"
	"context"
	"github.com/jfcg/sorty/v2"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
)

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

func Remove[T any](items []T, condition func(T) bool) []T {
	filteredItems := make([]T, 0)
	for _, item := range items {
		if !condition(item) {
			filteredItems = append(filteredItems, item)
		}
	}
	return filteredItems
}

func InsertAtIndex[T any](items []T, index int, value T) []T {
	if index == 0 {
		return append([]T{value}, items...)
	}
	if index == len(items) {
		return append(items, value)
	}
	temp := append(Copy(items)[:index], value)
	return append(temp, items[index:]...)
}

func RemoveIndex[T any](items []T, index int) []T {
	if index == 0 {
		return items[1:]
	}
	if index == len(items)-1 {
		return items[:len(items)-1]
	}
	return append(items[:index], items[index+1:]...)
}

func GetLast[T any](items []T) T {
	return items[len(items)-1]
}

func Filter[V any](values []V, condition func(V) bool) []V {
	filteredValues := make([]V, 0)
	for _, v := range values {
		if condition(v) {
			filteredValues = append(filteredValues, v)
		}
	}
	return filteredValues
}

func CompareArrays[T comparable](a, b []T) (bool, int) {
	if a == nil && b == nil {
		return true, -1
	}
	if a == nil || b == nil {
		return false, -1
	}
	if len(a) != len(b) {
		return false, -1
	}
	for i := range a {
		if a[i] != b[i] {
			return false, i
		}
	}
	return true, -1
}

func Sum(values []int) int {
	sum := 0
	for _, v := range values {
		sum += v
	}
	return sum
}

func MaxValue(values []int) int {
	m := values[0]
	for _, v := range values {
		if v > m {
			m = v
		}
	}
	return m
}

func GetValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func FillArray[T any](value T, numElements int) []T {
	if numElements <= 0 {
		return []T{}
	}
	values := make([]T, numElements)
	for i := 0; i < numElements; i++ {
		values[i] = value
	}
	return values
}

func Shuffle[T any](items []T) {
	for i := len(items) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		items[i], items[j] = items[j], items[i]
	}
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

func MapEntries[K comparable, V any, O any](m map[K]V, f func(K, V) O) []O {
	result := make([]O, 0, len(m))
	for k, v := range m {
		result = append(result, f(k, v))
	}
	return result
}

func Contains[T any](items []T, f func(T) bool) bool {
	for _, item := range items {
		if f(item) {
			return true
		}
	}
	return false
}

func DoesNotContain[T any](items []T, f func(T) bool) bool {
	return !Contains(items, f)
}

func Find[T any](items []T, defaultT T, f func(T) bool) (T, bool) {
	for _, item := range items {
		if f(item) {
			return item, true
		}
	}
	return defaultT, false
}

func FindInMap[K comparable, V any](m map[K]V, f func(K, V) bool, defaultKey K, defaultValue V) (K, V, bool) {
	for k, v := range m {
		if f(k, v) {
			return k, v, true
		}
	}
	return defaultKey, defaultValue, false
}

func FindKey[K comparable, V any](m map[K]V, f func(K, V) bool, defaultValue K) (K, bool) {
	for k, v := range m {
		if f(k, v) {
			return k, true
		}
	}
	return defaultValue, false
}

func FindValue[K comparable, V any](m map[K]V, f func(K, V) bool, defaultValue V) (V, bool) {
	for k, v := range m {
		if f(k, v) {
			return v, true
		}
	}
	return defaultValue, false
}

func DoesMapContain[K comparable, V any](m map[K]V, f func(K, V) bool) bool {
	for k, v := range m {
		if f(k, v) {
			return true
		}
	}
	return false
}

func DoesMapNotContain[K comparable, V any](m map[K]V, f func(K, V) bool) bool {
	return !DoesMapContain(m, f)
}

func FindIndex[T any](items []T, f func(T) bool) int {
	for i, item := range items {
		if f(item) {
			return i
		}
	}
	return -1
}

func FindLastIndex[T any](items []T, f func(T) bool) int {
	for i := len(items) - 1; i >= 0; i-- {
		if f(items[i]) {
			return i
		}
	}
	return -1
}

func Copy[T any](items []T) []T {
	result := make([]T, len(items))
	copy(result, items)
	return result
}

func CopyMap[K comparable, V any](m map[K]V) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func Swap[T any](items []T, i, j int) {
	items[i], items[j] = items[j], items[i]
}

func Flatten[T any](items [][]T) []T {
	var result []T
	for _, item := range items {
		result = append(result, item...)
	}
	return result
}

func FlatMap[T any, O any](items []T, f func(T) []O) []O {
	var result []O
	for _, item := range items {
		result = append(result, f(item)...)
	}
	return result
}

func Fold[T any, O any](items []T, initial O, f func(O, T) O) O {
	result := initial
	for _, item := range items {
		result = f(result, item)
	}
	return result
}

func Apply[T any](items []T, f func(T)) {
	for _, item := range items {
		f(item)
	}
}

func Unless[T any](items []T, f func(T) bool) bool {
	for _, item := range items {
		if !f(item) {
			return false
		}
	}
	return true
}

func ParallelFind[T any](items []T, f func(T) bool) *T {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure all paths cancel the context to avoid a context leak

	var found atomic.Value
	found.Store((*T)(nil)) // Initialize with nil

	var wg sync.WaitGroup
	numProcs := runtime.NumCPU() // Number of logical CPUs
	segmentSize := (len(items) + numProcs - 1) / numProcs

	for i := 0; i < len(items); i += segmentSize {
		end := i + segmentSize
		if end > len(items) {
			end = len(items)
		}

		wg.Add(1)
		go func(segment []T) {
			defer wg.Done()
			for _, item := range segment {
				select {
				case <-ctx.Done():
					return // Exit if context is cancelled
				default:
					if f(item) {
						found.Store(&item)
						cancel() // Cancel other goroutines
						return
					}
				}
			}
		}(items[i:end])
	}

	wg.Wait()
	result, _ := found.Load().(*T)
	return result
}

func Sort[T any](items []T, less func(T, T) bool) {
	// Define the Lesswap function required by sorty
	lesswap := func(i, k, r, s int) bool {
		if less(items[i], items[k]) {
			if r != s {
				items[r], items[s] = items[s], items[r]
			}
			return true
		}
		return false
	}

	// Call sorty.Sort with the length of the items and the lesswap function
	sorty.Sort(len(items), lesswap)
}

func SortOrdered[T cmp.Ordered](items []T) {
	Sort(items, func(a, b T) bool {
		return a < b
	})
}

func ParallelContains[T any](items []T, f func(T) bool) bool {
	return ParallelFind(items, f) != nil
}

func FindLast[T any](items []T, f func(T) bool) *T {
	for i := len(items) - 1; i >= 0; i-- {
		if f(items[i]) {
			return &items[i]
		}
	}
	return nil
}

func Count[T any](items []T, f func(T) bool) int {
	if items == nil {
		return 0
	}
	count := 0
	for _, item := range items {
		if f(item) {
			count++
		}
	}
	return count
}
