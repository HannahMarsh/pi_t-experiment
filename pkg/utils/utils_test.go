package utils

import (
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
		nTrue := Count(b, func(bb bool) bool {
			return bb
		})
		nFalse := Count(b, func(bb bool) bool {
			return !bb
		})
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
