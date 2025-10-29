package message

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
)

// PublicKeyToString converts solana.PublicKey to a base58 string
func PublicKeyToString(pubKey solana.PublicKey) string {
	return pubKey.String()
}

// StringToPublicKey converts a hex string back to xdr.PublicKey
func StringToPublicKey(s string) (solana.PublicKey, error) {
	if s == "" {
		return solana.PublicKey{}, nil
	}
	pubKey, err := solana.PublicKeyFromBase58(s)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("failed to decode hex string: %w", err)
	}
	return pubKey, nil
}
