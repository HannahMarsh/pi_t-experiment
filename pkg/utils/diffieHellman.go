package utils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"

	"github.com/HannahMarsh/PrettyLogger"
)

func generateDHKeyPair() (*big.Int, *big.Int, error) {
	// Use standard Diffie-Hellman parameters (could be replaced with your own)
	p, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1"+
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD"+
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245"+
		"E485B576625E7EC6F44C42E9A63A36210000000000090563", 16)
	g := big.NewInt(2) // A common choice for g
	privKey, err := rand.Int(rand.Reader, p)
	if err != nil {
		return nil, nil, err
	}
	pubKey := new(big.Int).Exp(g, privKey, p)
	return privKey, pubKey, nil
}

func generateECDHKeyPair() (*ecdsa.PrivateKey, crypto.PublicKey, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, PrettyLogger.WrapError(err, "failed to generate ECDSA key pair")
	}
	return privKey, privKey.PublicKey, nil
}
