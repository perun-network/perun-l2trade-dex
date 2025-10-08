package message

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/channel/types"
)

// SolanaAssetConfig represents the configuration for a Solana asset.
type SolanaAssetConfig struct {
	Code         string          `json:"code"`
	Name         string          `json:"name"`
	Type         SolanaAssetType `json:"type"`
	ContractID   string          `json:"contractID"`
	ContractPath string          `json:"contractPath"`
	ChainID      ChainID         `json:"chainID"`
	DeployerSeed string          `json:"deployerSeed"`
}

// ContractID is a type alias for string representing a contract ID.
type ContractID = string

// SolanaAssetType represents the type of Solana asset.
type SolanaAssetType uint

// SolanaAssetConfigMap is a map of Solana asset configurations.
type SolanaAssetConfigMap map[string]SolanaAssetConfig

// ReflectSolanaAssetType is the reflection type of the asset type used for parsing custom types.
var ReflectSolanaAssetType = reflect.TypeOf(SolanaAssetType(0))

// MakeContractID makes a ChainID for the given id.
func MakeContractID(id string) ContractID {
	return id
}

func (c SolanaAssetConfig) getCode() string {
	return c.Code
}

func (c SolanaAssetConfig) getName() string {
	return c.Name
}

func (c SolanaAssetConfig) getType() uint {
	return uint(c.Type)
}

const (
	// AssetTypeSOL represents a Solana asset.
	AssetTypeSOL SolanaAssetType = iota
	// AssetTypeSPL represents a SPL token.
	AssetTypeSPL
)

var solanaAssetTypeNames = []string{"SOL", "SPL"}

func (t SolanaAssetType) String() string {
	return solanaAssetTypeNames[t]
}

// MarshalJSON marshals SolanaAssetType into JSON.
func (t SolanaAssetType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON unmarshals SolanaAssetType from JSON.
func (t *SolanaAssetType) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*t, err = ParseStellarAssetType(s)
	return err
}

// ParseStellarAssetType parses an asset type string.
func ParseStellarAssetType(s string) (SolanaAssetType, error) {
	for i, assType := range stellarAssetTypeNames {
		if s == assType {
			return StellarAssetType(i), nil
		}
	}
	log.Println("Invalid value for asset type", s)
	return StellarAssetType(0), errors.New("invalid value for asset type")
}

// StellarAsset represents a Stellar asset.
type StellarAsset struct {
	ContractID string `json:"contractID"`
}

// AssetType returns the type of the asset.
func (a StellarAsset) AssetType() string {
	return "Stellar"
}

// Code returns the unique identifier of the asset.
func (a StellarAsset) Code() string {
	// Use the contract ID as the unique identifier for the Stellar asset.
	return a.ContractID
}

// MarshalJSON marshals StellarAsset into JSON.
func (a StellarAsset) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		AssetType  string `json:"assetType"`
		ContractID string `json:"contractID"`
	}{
		AssetType:  "Stellar", // To identify asset type
		ContractID: a.ContractID,
	})
}

// UnmarshalJSON unmarshals StellarAsset from JSON.
func (a *StellarAsset) UnmarshalJSON(data []byte) error {
	var raw struct {
		ContractID string `json:"contractID"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	a.ContractID = raw.ContractID
	return nil
}

// StellarAssetsToAssets converts a slice of StellarAsset to a slice of Asset.
func StellarAssetsToAssets(in []StellarAsset) []Asset {
	out := make([]Asset, len(in))
	for i, asset := range in {
		out[i] = &asset
	}
	return out
}

// NewStellarAsset creates a new Asset from a StellarAssetConfig.
func NewStellarAsset(a StellarAssetConfig) (*StellarAsset, error) {
	fmt.Println("Creating new stellar asset", a.ContractID)

	return &StellarAsset{
		ContractID: a.ContractID,
	}, nil
}

// MakeStellarAssets creates a slice of Assets from a slice of AssetConfig.
func MakeStellarAssets(a types.StellarAsset) StellarAsset {
	return StellarAsset{ContractID: a.Asset.ContractID().HexString()}
}

// MakeStellarAssetsGP creates a slice of StellarAsset from a slice of channel.Asset.
func MakeStellarAssetsGP(a []channel.Asset) []StellarAsset {
	out := make([]StellarAsset, len(a))
	for i, asset := range a {
		out[i] = MakeStellarAssets(*asset.(*types.StellarAsset))
	}
	return out
}

// MakePerunStellarAssets creates a slice of Assets from a slice of AssetConfig.
func MakePerunStellarAssets(in []StellarAsset) ([]channel.Asset, error) {
	out := make([]channel.Asset, len(in))
	for i, asset := range in {
		hash, err := StringToHash(asset.ContractID)
		if err != nil {
			return nil, err
		}
		add, err := types.MakeContractAddress(hash)
		if err != nil {
			return nil, err
		}
		out[i], _ = types.NewStellarAssetFromScAddress(add)
	}
	return out, nil
}

// MakeStellarAssetsFromAssets creates a slice of Assets from a slice of AssetConfig.
func MakeStellarAssetsFromAssets(in []Asset) []StellarAsset {
	out := make([]StellarAsset, len(in))
	for i, asset := range in {
		out[i] = *(asset.(*StellarAsset))
	}
	return out
}

// MakeAssetSlice creates a slice of AssetConfig from a slice of StellarAssetConfig.
func MakeAssetSlice(arr []StellarAssetConfig) []AssetConfig {
	am := make([]AssetConfig, len(arr))
	for id, a := range arr {
		am[id] = a
	}
	return am
}

// StellarAssetConfigFromAssetConfig creates a slice of StellarAssetConfig from a slice of AssetConfig.
func StellarAssetConfigFromAssetConfig(arr []AssetConfig) []StellarAssetConfig {
	am := make([]StellarAssetConfig, len(arr))
	for id, a := range arr {
		am[id] = a.(StellarAssetConfig)
	}
	return am
}

// StellarAssetMapToArray converts an AssetConfigMap to a slice of AssetConfig.
func StellarAssetMapToArray(am StellarAssetConfigMap) []StellarAssetConfig {
	var arr []StellarAssetConfig
	for _, a := range am {
		arr = append(arr, a)
	}
	return arr
}

// MakeStellarAssetMap creates a StellarAssetConfigMap from a slice of StellarAssetConfig.
func MakeStellarAssetMap(arr []StellarAssetConfig) StellarAssetConfigMap {
	am := make(StellarAssetConfigMap)
	for _, a := range arr {
		am[a.Code] = a
	}
	return am
}

// StellarAssetArrayToMap converts an AssetConfigMap to a slice of AssetConfig.
func StellarAssetArrayToMap(am []StellarAssetConfig) StellarAssetConfigMap {
	assetMap := StellarAssetConfigMap{}
	for _, a := range am {
		assetMap[a.ContractID] = a
	}
	return assetMap
}

// StringToScAddress converts a string to an xdr.ScAddress.
func StringToScAddress(s string) (xdr.ScAddress, error) {
	hash, err := StringToHash(s)
	if err != nil {
		return xdr.ScAddress{}, err
	}
	return types.MakeContractAddress(hash)
}

// ConvertAssetConfigMapToScVec converts a StellarAssetConfigMap to a slice of xdr.ScAddress.
func ConvertAssetConfigMapToScVec(addresses StellarAssetConfigMap) ([]xdr.ScAddress, error) {
	var scVec []xdr.ScAddress

	for _, addr := range addresses {
		tokenAddr, err := StringToScAddress(addr.ContractID)
		if err != nil {
			return nil, err
		}

		// Append the xdr.ScVal to the xdr.ScVec
		scVec = append(scVec, tokenAddr)
	}

	return scVec, nil
}

// ConvertAddressesToScVec converts a slice of xdr.ScAddress to an xdr.ScVec.
func ConvertAddressesToScVec(addresses []xdr.ScAddress) xdr.ScVec {
	var scVec xdr.ScVec

	for _, addr := range addresses {
		// Convert xdr.ScAddress to xdr.ScVal
		addrScVal := xdr.ScVal{
			Type:    xdr.ScValTypeScvAddress,
			Address: &addr,
		}

		// Append the xdr.ScVal to the xdr.ScVec
		scVec = append(scVec, addrScVal)
	}

	return scVec
}
