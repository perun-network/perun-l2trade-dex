package message

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// BigInt is a wrapper around big.Int that implements the json.Marshaler and json.Unmarshaler interfaces.
type BigInt struct {
	*big.Int
}

// MakeBigInt creates a BigInt from a big.Int.
func MakeBigInt(b *big.Int) BigInt {
	return BigInt{new(big.Int).Set(b)}
}

// MarshalJSON marshals a BigInt into JSON.
func (b BigInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.Int.String())
}

// UnmarshalJSON unmarshals a BigInt from JSON.
func (b *BigInt) UnmarshalJSON(d []byte) error {
	var s string
	if err := json.Unmarshal(d, &s); err != nil {
		return err
	}
	var ok bool
	b.Int, ok = new(big.Int).SetString(s, 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", d)
	}
	return nil
}
