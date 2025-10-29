package message

import (
	"encoding/json"
	"reflect"

	"github.com/gagliardetto/solana-go"
	schannel "github.com/perun-network/perun-solana-backend/channel"
	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
)

// SolanaAssetConfig represents the configuration for a Solana asset.
type SolanaAssetConfig struct {
	Code    string          `json:"code"`
	Name    string          `json:"name"`
	Type    SolanaAssetType `json:"type"`
	ChainID ChainID         `json:"chainID"`
	Mint    Mint            `json:"mint"`
}

// Mint is a type alias for string representing a mint address.
type Mint = string

// SolanaAssetType represents the type of Solana asset.
type SolanaAssetType uint

// SolanaAssetConfigMap is a map of Solana asset configurations.
type SolanaAssetConfigMap map[string]SolanaAssetConfig

// ReflectSolanaAssetType is the reflection type of the asset type used for parsing custom types.
var ReflectSolanaAssetType = reflect.TypeOf(SolanaAssetType(0))

// MakeMint makes a Mint for the given id.
func MakeMint(id string) Mint {
	return Mint(id)
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
	*t, err = ParseSolanaAssetType(s)
	return err
}

// ParseSolanaAssetType parses an asset type string.
func ParseSolanaAssetType(s string) (SolanaAssetType, error) {
	for i, assType := range solanaAssetTypeNames {
		if s == assType {
			return SolanaAssetType(i), nil
		}
	}
	return SolanaAssetType(0), errors.New("invalid value for asset type")
}

// SolanaAsset represents a Solana asset.
type SolanaAsset struct {
	Mint string `json:"mint"`
}

// AssetType returns the type of the asset.
func (a SolanaAsset) AssetType() string {
	return "Solana"
}

// Code returns the unique identifier of the asset.
func (a SolanaAsset) Code() string {
	// Use the mint address as the unique identifier for the Solana asset.
	return a.Mint
}

// MarshalJSON marshals SolanaAsset into JSON.
func (a SolanaAsset) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		AssetType string `json:"assetType"`
		Mint      string `json:"mint"`
	}{
		AssetType: "Solana", // To identify asset type
		Mint:      a.Mint,
	})
}

// UnmarshalJSON unmarshals SolanaAsset from JSON.
func (a *SolanaAsset) UnmarshalJSON(data []byte) error {
	var raw struct {
		Mint string `json:"mint"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	a.Mint = raw.Mint
	return nil
}

// SolanaAssetsToAssets converts a slice of SolanaAsset to a slice of Asset.
func SolanaAssetsToAssets(in []SolanaAsset) []Asset {
	out := make([]Asset, len(in))
	for i, asset := range in {
		out[i] = &asset
	}
	return out
}

// NewSolanaAsset creates a new Asset from a SolanaAssetConfig.
func NewSolanaAsset(a SolanaAssetConfig) (*SolanaAsset, error) {
	return &SolanaAsset{
		Mint: a.Mint,
	}, nil
}

// MakeSolanaAssets creates a slice of Assets from a slice of AssetConfig.
func MakeSolanaAssets(a schannel.SolanaCrossAsset) SolanaAsset {
	if a.Asset.IsSOL {
		return SolanaAsset{Mint: ""}
	}
	return SolanaAsset{Mint: a.Asset.Mint.String()}
}

// MakeSolanaAssetsGP creates a slice of SolanaAsset from a slice of channel.Asset.
func MakeSolanaAssetsGP(a []channel.Asset) []SolanaAsset {
	out := make([]SolanaAsset, len(a))
	for i, asset := range a {
		out[i] = MakeSolanaAssets(*asset.(*schannel.SolanaCrossAsset))
	}
	return out
}

// MakePerunSolanaAssets creates a slice of Assets from a slice of AssetConfig.
func MakePerunSolanaAssets(in []SolanaAsset) ([]channel.Asset, error) {
	out := make([]channel.Asset, len(in))
	for i, asset := range in {
		mint, err := StringToPublicKey(asset.Mint)
		if err != nil {
			return nil, err
		}
		out[i] = schannel.NewSolanaCrossAssetFromMint(mint)
	}
	return out, nil
}

// MakeSolanaAssetsFromAssets creates a slice of Assets from a slice of AssetConfig.
func MakeSolanaAssetsFromAssets(in []Asset) []SolanaAsset {
	out := make([]SolanaAsset, len(in))
	for i, asset := range in {
		out[i] = *(asset.(*SolanaAsset))
	}
	return out
}

// MakeAssetSlice creates a slice of AssetConfig from a slice of SolanaAssetConfig.
func MakeAssetSlice(arr []SolanaAssetConfig) []AssetConfig {
	am := make([]AssetConfig, len(arr))
	for id, a := range arr {
		am[id] = a
	}
	return am
}

// SolanaAssetConfigFromAssetConfig creates a slice of SolanaAssetConfig from a slice of AssetConfig.
func SolanaAssetConfigFromAssetConfig(arr []AssetConfig) []SolanaAssetConfig {
	am := make([]SolanaAssetConfig, len(arr))
	for id, a := range arr {
		am[id] = a.(SolanaAssetConfig)
	}
	return am
}

// SolanaAssetMapToArray converts an AssetConfigMap to a slice of AssetConfig.
func SolanaAssetMapToArray(am SolanaAssetConfigMap) []SolanaAssetConfig {
	var arr []SolanaAssetConfig
	for _, a := range am {
		arr = append(arr, a)
	}
	return arr
}

// MakeSolanaAssetMap creates a SolanaAssetConfigMap from a slice of SolanaAssetConfig.
func MakeSolanaAssetMap(arr []SolanaAssetConfig) SolanaAssetConfigMap {
	am := make(SolanaAssetConfigMap)
	for _, a := range arr {
		am[a.Code] = a
	}
	return am
}

// SolanaAssetArrayToMap converts an AssetConfigMap to a slice of AssetConfig.
func SolanaAssetArrayToMap(am []SolanaAssetConfig) SolanaAssetConfigMap {
	assetMap := SolanaAssetConfigMap{}
	for _, a := range am {
		assetMap[a.Mint] = a
	}
	return assetMap
}

// StringToSolanaPublicKey converts a string to a solana.PublicKey.
func StringToSolanaPublicKey(s string) (solana.PublicKey, error) {
	if s == "" {
		return solana.PublicKey{}, nil
	}

	pubKey, err := solana.PublicKeyFromBase58(s)
	if err != nil {
		return solana.PublicKey{}, err
	}
	return pubKey, nil
}

// ConvertAssetConfigMapToVec converts a SolanaAssetConfigMap to a slice of solana.PublicKey.
func ConvertAssetConfigMapToVec(addresses SolanaAssetConfigMap) ([]solana.PublicKey, error) {
	var vec []solana.PublicKey

	for _, addr := range addresses {
		tokenAddr, err := StringToSolanaPublicKey(addr.Mint)
		if err != nil {
			return nil, err
		}

		vec = append(vec, tokenAddr)
	}

	return vec, nil
}
