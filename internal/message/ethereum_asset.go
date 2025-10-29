package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	echannel "github.com/perun-network/perun-eth-backend/channel"

	"perun.network/go-perun/channel"
)

type (
	// EthereumAssetConfig represents the configuration for an Ethereum asset.
	EthereumAssetConfig struct {
		Code        string            `json:"code"`
		Name        string            `json:"name"`
		Type        EthereumAssetType `json:"type"`
		ChainID     ChainID           `json:"chainID"`
		Address     common.Address    `json:"address"`
		AssetHolder common.Address    `json:"assetHolder"`
	}

	// EthereumAssetCode is the code of an Ethereum asset.
	EthereumAssetCode string
	// EthereumAssetConfigMap is a map of Ethereum asset configurations.
	EthereumAssetConfigMap map[string]EthereumAssetConfig
	// EthereumAssetType is the type of an Ethereum asset.
	EthereumAssetType uint
)

// ReflectEthereumAssetType is the reflection type of the asset type used for parsing
// custom types.
var ReflectEthereumAssetType = reflect.TypeOf(EthereumAssetType(0))

func (c EthereumAssetConfig) getCode() string {
	return c.Code
}

func (c EthereumAssetConfig) getName() string {
	return c.Name
}

func (c EthereumAssetConfig) getType() uint {
	return uint(c.Type)
}

const (
	// AssetTypeETH represents an Ethereum asset.
	AssetTypeETH EthereumAssetType = iota
	// AssetTypeERC20 represents an ERC20 token.
	AssetTypeERC20
)

var ethereumAssetTypeNames = []string{"ETH", "ERC20"}

func (t EthereumAssetType) String() string {
	return ethereumAssetTypeNames[t]
}

// MarshalJSON marshals an EthereumAssetType into JSON.
func (t EthereumAssetType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON unmarshals an EthereumAssetType from JSON.
func (t *EthereumAssetType) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*t, err = ParseEthereumAssetType(s)
	return err
}

// ParseEthereumAssetType parses a string to an EthereumAssetType.
func ParseEthereumAssetType(s string) (EthereumAssetType, error) {
	for i, assType := range ethereumAssetTypeNames {
		if s == assType {
			return EthereumAssetType(i), nil
		}
	}

	err := fmt.Errorf("invalid value for asset type. The value is '%s',"+
		" but must be one of '%v'", s, ethereumAssetTypeNames)

	return EthereumAssetType(0), err
}

// EthereumAsset Asset is a JSON encodable representation of echannel.Asset.
type EthereumAsset struct {
	AssetHolder common.Address `json:"assetHolder"`
	ChainID     ChainID        `json:"chainID"`
}

// AssetType returns the type of the asset.
func (a EthereumAsset) AssetType() string {
	return "Ethereum"
}

// Code Implements method for EthereumAsset.
func (a EthereumAsset) Code() string {
	// Use a unique identifier based on the Ethereum asset properties.
	return a.AssetHolder.String() + a.ChainID.String()
}

// MarshalJSON marshals EthereumAsset into JSON.
func (a EthereumAsset) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		AssetType   string `json:"assetType"`
		AssetHolder string `json:"assetHolder"`
		ChainID     string `json:"chainID"`
	}{
		AssetType:   "Ethereum",          // To identify asset type
		AssetHolder: a.AssetHolder.Hex(), // Hex-encoded address
		ChainID:     a.ChainID.String(),  // Convert ChainID (big.Int) to string
	})
}

// UnmarshalJSON unmarshals EthereumAsset from JSON.
func (a *EthereumAsset) UnmarshalJSON(data []byte) error {
	var raw struct {
		AssetHolder string `json:"assetHolder"`
		ChainID     string `json:"chainID"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Convert the hex string back to common.EthAddress
	a.AssetHolder = common.HexToAddress(raw.AssetHolder)

	// Parse ChainID from string to big.Int (ChainID type)
	chainID := new(big.Int)
	if _, ok := chainID.SetString(raw.ChainID, 10); !ok {
		return fmt.Errorf("invalid chainID: %s", raw.ChainID)
	}
	a.ChainID = MakeChainID(chainID)

	return nil
}

// NewChainIDFromString converts a string to a ChainID (*big.Int).
// The string can be in either decimal or hexadecimal format.
func NewChainIDFromString(s string) (ChainID, error) {
	chainID := new(big.Int)

	// Try parsing the string as a decimal or hexadecimal number.
	// "0x" prefix indicates hexadecimal format.
	_, ok := chainID.SetString(s, 0) // `0` automatically detects base (10 or 16).
	if !ok {
		return ChainID{}, errors.New("invalid ChainID format")
	}

	return ChainID{chainID}, nil
}

// EthereumAssetConfigToAssetConfig converts a slice of EthereumAssetConfig to a slice of AssetConfig.
func EthereumAssetConfigToAssetConfig(assets []AssetConfig) []EthereumAssetConfig {
	var ethereumAssets []EthereumAssetConfig
	for _, asset := range assets {
		ethereumAssets = append(ethereumAssets, asset.(EthereumAssetConfig))
	}
	return ethereumAssets
}

// NewEthereumAsset creates a new EthereumAsset from an echannel.Asset.
func NewEthereumAsset(a echannel.Asset) EthereumAsset {
	chainID := new(big.Int)

	// Set the big.Int from the string (the second argument is the base, e.g., 10 for decimal)
	_, success := chainID.SetString(string(a.LedgerBackendID().LedgerID().MapKey()), 10)
	if !success {
		return EthereumAsset{}
	}
	return EthereumAsset{
		AssetHolder: a.EthAddress(),
		ChainID:     MakeChainID(chainID),
	}
}

// MakeEthereumAssets creates a slice of Asset from a slice of channel.Asset.
func MakeEthereumAssets(in []channel.Asset) []EthereumAsset {
	out := make([]EthereumAsset, len(in))
	for i, asset := range in {
		out[i] = NewEthereumAsset(*(asset.(*echannel.Asset)))
	}
	return out
}

// GeneralizeEthAssets generalizes a slice of ethereum assets.
func GeneralizeEthAssets(in []EthereumAsset) []Asset {
	out := make([]Asset, len(in))
	for i, asset := range in {
		out[i] = &asset
	}
	return out
}

// MakePerunEthereumAssets creates a slice of channel.Asset from a slice of Asset.
func MakePerunEthereumAssets(in []EthereumAsset) []channel.Asset {
	out := make([]channel.Asset, len(in))
	for i, asset := range in {
		out[i] = echannel.NewAsset(asset.ChainID.Int, asset.AssetHolder)
	}
	return out
}

// MakeEthereumAssetMap converts a slice of EthereumAssetConfig to an EthereumAssetConfigMap.
func MakeEthereumAssetMap(arr []EthereumAssetConfig) EthereumAssetConfigMap {
	am := make(EthereumAssetConfigMap)
	for _, a := range arr {
		am[a.Code] = a
	}
	return am
}

// EthereumAssetArrayToArray converts a slice of EthereumAssetConfig to a slice of AssetConfig.
func EthereumAssetArrayToArray(am []EthereumAssetConfig) []AssetConfig {
	var arr []AssetConfig
	for _, a := range am {
		arr = append(arr, a)
	}
	return arr
}

// EthereumAssetMapToArray converts an EthereumAssetConfigMap to a slice of EthereumAssetConfig.
func EthereumAssetMapToArray(am EthereumAssetConfigMap) []EthereumAssetConfig {
	var arr []EthereumAssetConfig
	for _, a := range am {
		arr = append(arr, a)
	}
	return arr
}
