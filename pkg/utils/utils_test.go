package utils

import (
	"reflect"
	"testing"
)

func TestGeneratePermutations(t *testing.T) {
	numTrue, numFalse := 3, 2
	arr := GenerateUniquePermutations(numTrue, numFalse)
	if Contains(arr, func(b []bool) bool {
		return len(b) != numTrue+numFalse
	}) {
		t.Fatalf("not all arrays are length %d", numTrue+numFalse)
	}
	if Contains(arr, func(b []bool) bool {
		nTrue := Count(b, true)
		nFalse := Count(b, false)
		return nTrue != numTrue || nFalse != numFalse
	}) {
		t.Fatalf("not all arrays have the right number of true false")
	}

	for i := 0; i < len(arr); i++ {
		for j := i + 1; j < len(arr); j++ {
			c, _ := CompareArrays(arr[i], arr[j])
			if c {
				t.Fatalf("arrays at indexes %d and %d are not unique", i, j)
			}
		}
	}
	x := Factorial(numFalse+numTrue) / (Factorial(numFalse) * Factorial(numTrue))
	if len(arr) != x {
		t.Fatalf("Expected length %d=(%d+%d)!/%d!%d! but got %d", x, numFalse, numTrue, numFalse, numTrue, len(arr))
	}
}

func TestDeterministicShuffle(t *testing.T) {
	// Test data
	originalArr := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	shuffled1 := Copy(originalArr)
	shuffled2 := Copy(originalArr)

	// Test with a fixed seed
	seed := int64(42)
	DeterministicShuffle(shuffled1, seed)
	DeterministicShuffle(shuffled2, seed)

	// The shuffled array should be different from the original
	if reflect.DeepEqual(originalArr, shuffled1) || reflect.DeepEqual(originalArr, shuffled2) {
		t.Errorf("Expected arrays to be shuffled, but they were not")
	}

	if !reflect.DeepEqual(shuffled1, shuffled2) {
		t.Errorf("Expected array to be shuffled in the same way with the same seed, but it was not")
	}

	// test od different seed
	shuffled3 := Copy(originalArr)
	seed = int64(43)

	DeterministicShuffle(shuffled3, seed)

	// The shuffled array should be different from the original
	if reflect.DeepEqual(originalArr, shuffled3) {
		t.Errorf("Expected array to be shuffled, but it was not")
	}

	if reflect.DeepEqual(shuffled1, shuffled3) {
		t.Errorf("Expected array to be shuffled in a different way with a different seed, but it was not")
	}
}
