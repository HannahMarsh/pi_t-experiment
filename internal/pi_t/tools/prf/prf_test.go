package prf

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"golang.org/x/exp/slog"
	"log"
	"testing"
)

func TestPRF_F1(t *testing.T) {
	num0s := 0
	num1s := 0
	for i := 0; i < 100; i++ {
		privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
		if err != nil {
			t.Fatalf("KeyGen() error: %v", err)
		}
		privateKeyPEM1, publicKeyPEM1, err := keys.KeyGen()
		if err != nil {
			t.Fatalf("KeyGen() error: %v", err)
		}
		scalar, err := keys.GenerateScalar()
		if err != nil {
			slog.Error("failed to generate scalar", err)
			t.Fatalf("GenerateScalar() error: %v", err)
		}

		hopIndex := 1
		expectedResult := computeExpectedPRF_F1(privateKeyPEM, publicKeyPEM1, scalar, hopIndex)
		actualResult := PRF_F1(privateKeyPEM, publicKeyPEM1, scalar, hopIndex)

		if expectedResult != actualResult {
			t.Errorf("PRF_F1() = %d; want %d", actualResult, expectedResult)
		}

		expectedResult2 := computeExpectedPRF_F1(privateKeyPEM1, publicKeyPEM, scalar, hopIndex)
		actualResult2 := PRF_F1(privateKeyPEM1, publicKeyPEM, scalar, hopIndex)

		if expectedResult2 != actualResult2 {
			t.Errorf("PRF_F1() = %d; want %d", actualResult2, expectedResult2)
		}

		if actualResult != actualResult2 {
			t.Errorf("PRF_F1() = %d; want %d", actualResult, actualResult2)
		}

		if actualResult == 0 {
			num0s++
		} else {
			num1s++
		}
		if actualResult2 == 0 {
			num0s++
		} else {
			num1s++
		}
	}

	fmt.Println("Number of 0s: ", num0s)
	fmt.Println("Number of 1s: ", num1s)
}

func TestPRF_F2(t *testing.T) {
	for i := 0; i < 100; i++ {
		privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
		if err != nil {
			t.Fatalf("KeyGen() error: %v", err)
		}
		privateKeyPEM1, publicKeyPEM1, err := keys.KeyGen()
		if err != nil {
			t.Fatalf("KeyGen() error: %v", err)
		}
		scalar, err := keys.GenerateScalar()
		if err != nil {
			slog.Error("failed to generate scalar", err)
			t.Fatalf("GenerateScalar() error: %v", err)
		}

		hopIndex := 1
		expectedResult := computeExpectedPRF_F2(privateKeyPEM, publicKeyPEM1, scalar, hopIndex)
		actualResult := PRF_F2(privateKeyPEM, publicKeyPEM1, scalar, hopIndex)

		if !bytes.Equal(expectedResult, actualResult) {
			t.Errorf("PRF_F2() = %x; want %x", actualResult, expectedResult)
		}

		expectedResult2 := computeExpectedPRF_F2(privateKeyPEM1, publicKeyPEM, scalar, hopIndex)
		actualResult2 := PRF_F2(privateKeyPEM1, publicKeyPEM, scalar, hopIndex)

		if !bytes.Equal(expectedResult2, actualResult2) {
			t.Errorf("PRF_F2() = %x; want %x", actualResult2, expectedResult2)
		}

		if string(actualResult) != string(actualResult2) {
			t.Errorf("PRF_F2() = %d; want %d", actualResult, actualResult2)
		}
	}
}

func computeExpectedPRF_F1(privateKeyPEM, publicKeyPEM string, scalar []byte, j int) int {
	sharedKey, err := keys.ComputeSharedKeyWithScalar(privateKeyPEM, publicKeyPEM, scalar)
	if err != nil {
		log.Fatalf("failed to compute shared key: %v", err)
	}
	h := hmac.New(sha256.New, sharedKey)
	binary.Write(h, binary.BigEndian, int64(j))
	result := h.Sum(nil)
	return int(result[0]) % 2 // Returns 0 or 1
}

func computeExpectedPRF_F2(privateKeyPEM, publicKeyPEM string, scalar []byte, j int) []byte {
	sharedKey, err := keys.ComputeSharedKeyWithScalar(privateKeyPEM, publicKeyPEM, scalar)
	if err != nil {
		log.Fatalf("failed to compute shared key: %v", err)
	}
	h := hmac.New(sha256.New, sharedKey)
	binary.Write(h, binary.BigEndian, int64(j))
	return h.Sum(nil)[:16] // Return the first 16 bytes as the nonce
}
