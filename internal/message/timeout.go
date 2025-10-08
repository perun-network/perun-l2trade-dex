package message

import (
	"encoding/json"
	"fmt"
)

// TimeoutType is the type of a timeout.
type TimeoutType uint

const (
	// DefaultTimeout is the default timeout type.
	DefaultTimeout TimeoutType = iota
	// HandleTimeout is the timeout type for handling a message.
	HandleTimeout
	// FundTimeout is the timeout type for funding.
	FundTimeout
	// SettleTimeout is the timeout type for settling.
	SettleTimeout
)

var timeoutTypeNames = []string{"Default", "Handle", "Funding", "Settle"}

// String returns the string representation of the timeout type.
func (t TimeoutType) String() string {
	return timeoutTypeNames[t]
}

// MarshalJSON marshals a TimeoutType into JSON.
func (t TimeoutType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON unmarshals a TimeoutType from JSON.
func (t *TimeoutType) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*t, err = ParseTimeoutType(s)
	return err
}

// ParseTimeoutType parses a timeout type string.
func ParseTimeoutType(s string) (TimeoutType, error) {
	for i, timeoutType := range timeoutTypeNames {
		if s == timeoutType {
			return TimeoutType(i), nil
		}
	}

	err := fmt.Errorf("invalid value for timeout type. The value is '%s',"+
		" but must be one of '%v'", s, timeoutTypeNames)

	return TimeoutType(0), err
}
