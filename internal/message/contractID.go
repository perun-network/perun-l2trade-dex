package message

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/stellar/go/xdr"
)

// HashToString converts xdr.Hash to a hex string
func HashToString(hash xdr.Hash) string {
	return hex.EncodeToString(hash[:])
}

// StringToHash converts a hex string back to xdr.Hash
func StringToHash(s string) (xdr.Hash, error) {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return xdr.Hash{}, fmt.Errorf("failed to decode hex string: %w", err)
	}
	var hash xdr.Hash
	copy(hash[:], bytes)
	return hash, nil
}

// GenerateRandomHash generates a random xdr.hash.
func GenerateRandomHash() (xdr.Hash, error) {
	var hash xdr.Hash
	_, err := rand.Read(hash[:])
	if err != nil {
		return xdr.Hash{}, fmt.Errorf("failed to generate random hash: %w", err)
	}
	return hash, nil
}
