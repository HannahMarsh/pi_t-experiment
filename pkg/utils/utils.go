package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slog"
	"io"
	rng "math/rand"
	"os"
	"sort"
	"time"
)

// ref: https://www.thorsten-hans.com/check-if-application-is-running-in-docker-container/
func IsRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return false
	}

	return true
}

func GenerateUniqueHash() string {
	// Create a buffer for random data
	randomData := make([]byte, 32) // 32 bytes for a strong unique value

	// Fill the buffer with random data
	if _, err := io.ReadFull(rand.Reader, randomData); err != nil {
		slog.Error("Failed to generate random data", err)
		return ""
	}

	// Create a new SHA256 hash
	hash := sha256.New()

	// Write the random data to the hash
	hash.Write(randomData)

	// Get the resulting hash as a byte slice
	hashBytes := hash.Sum(nil)

	// Encode the hash to a hexadecimal string
	hashString := hex.EncodeToString(hashBytes)

	return hashString
}

func GenerateRandomString(str string) string {
	length := len(str)
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err == nil {
		length = len(decoded)
	}
	randomBytes := make([]byte, 0)

	for len(randomBytes) < length {

		// Create a buffer for random data
		randomData := make([]byte, 32) // 32 bytes for a strong unique value

		// Fill the buffer with random data
		if _, err := io.ReadFull(rand.Reader, randomData); err != nil {
			slog.Error("Failed to generate random data", err)
			return ""
		}

		// Create a new SHA256 hash
		hash := sha256.New()

		// Write the random data to the hash
		hash.Write(randomData)

		// Get the resulting hash as a byte slice
		hashBytes := hash.Sum(nil)
		randomBytes = append(randomBytes, hashBytes...)
	}
	// Encode the hash to a hexadecimal string
	hashString := base64.StdEncoding.EncodeToString(randomBytes[:length])

	return hashString
}

func DropLastElement[T any](elements []T) []T {
	if elements == nil || len(elements) <= 1 {
		return []T{}
	}
	return elements[:len(elements)-1]
}

func DropFirstElement[T any](elements []T) []T {
	if elements == nil || len(elements) <= 1 {
		return []T{}
	}
	return elements[1:]
}

func DropFromLeft[T any](elements []T, n int) []T {
	if elements == nil || len(elements) <= 1 || n >= len(elements) {
		return []T{}
	}
	return elements[n:]
}

func DropFromRight[T any](elements []T, n int) []T {
	if elements == nil || len(elements) <= 1 || n >= len(elements) {
		return []T{}
	}
	return elements[:len(elements)-n]
}

var r = rng.New(rng.NewSource(time.Now().UnixNano()))

func RandomElement[T any](elements []T) (element T, index int) {
	index = r.Intn(len(elements))
	return elements[index], index
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Swap function to swap two elements in the array
func swap(arr []bool, i, j int) {
	arr[i], arr[j] = arr[j], arr[i]
}

// Recursive function to generate all unique permutations
func generatePermutations(arr []bool, start int, result *[][]bool) {
	if start == len(arr)-1 {
		perm := make([]bool, len(arr))
		copy(perm, arr)
		*result = append(*result, perm)
		return
	}
	seen := make(map[bool]bool)
	for i := start; i < len(arr); i++ {
		if seen[arr[i]] {
			continue
		}
		seen[arr[i]] = true
		swap(arr, start, i)
		generatePermutations(arr, start+1, result)
		swap(arr, start, i) // backtrack
	}
}

func ContainsElement[T comparable](elements []T, element T) bool {
	for _, e := range elements {
		if e == element {
			return true
		}
	}
	return false
}

// Function to create an array with a specified number of true and false values
func createInitialArray(numTrue, numFalse int) []bool {
	arr := make([]bool, numTrue+numFalse)
	for i := 0; i < numTrue; i++ {
		arr[i] = true
	}
	for i := numTrue; i < numTrue+numFalse; i++ {
		arr[i] = false
	}
	return arr
}

// Function to generate all unique permutations of an array with numTrue true values and numFalse false values
func GenerateUniquePermutations(numTrue, numFalse int) [][]bool {
	arr := createInitialArray(numTrue, numFalse)
	sort.Slice(arr, func(i, j int) bool { return !arr[i] && arr[j] })
	var result [][]bool
	generatePermutations(arr, 0, &result)
	return result
}

func Factorial(number int) int {

	// if the number has reached 1 then we have to
	// return 1 as 1 is the minimum value we have to multiply with
	if number == 1 {
		return 1
	}

	// multiplying with the current number and calling the function
	// for 1 lesser number
	factorialOfNumber := number * Factorial(number-1)

	// return the factorial of the current number
	return factorialOfNumber
}
