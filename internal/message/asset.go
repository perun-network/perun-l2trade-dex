package message

import (
	"math/big"

	echannel "github.com/perun-network/perun-eth-backend/channel"
	schannel "github.com/perun-network/perun-solana-backend/channel"
	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
)

// EthereumIndex represents the index for Ethereum assets.
const EthereumIndex = 1

// SolanaIndex represents the index for Solana assets.
const SolanaIndex = 6

// ChainID is a type alias for BigInt representing a chain ID.
type ChainID = BigInt

// AssetType represents the type of an asset.
type AssetType interface {
	String()
	MarshalJSON()
	UnmarshalJSON()
}

// Asset represents a generic asset.
type Asset interface {
	Code() string
	AssetType() string
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

// AssetConfig represents the configuration for an asset.
type AssetConfig interface {
	getCode() string
	getName() string
	getType() uint
}

// AssetConfigMap is a map of asset configurations.
type AssetConfigMap map[string]AssetConfig

// MakeAssetsGPAsAssets converts a slice of channel.Asset to a slice of Asset.
func MakeAssetsGPAsAssets(a []channel.Asset) []Asset {
	out := make([]Asset, len(a))
	for i, asset := range a {
		sa, ok := asset.(*schannel.SolanaCrossAsset)
		if ok {
			as := MakeSolanaAssets(*sa)
			out[i] = &as
		} else {
			ea, ok := asset.(*echannel.Asset)
			if ok {
				as := NewEthereumAsset(*ea)
				out[i] = &as
			}
		}
	}
	return out
}

// MakeChainID creates a ChainID from a big.Int.
func MakeChainID(i *big.Int) ChainID {
	return MakeBigInt(i)
}

// ToEthChainID converts a ChainID to an Ethereum chain ID.
func (i ChainID) ToEthChainID() echannel.ChainID {
	return echannel.MakeChainID(i.Int)
}

// MapKey converts a ChainID to a multi.LedgerIDMapKey.
func (i ChainID) MapKey() multi.LedgerIDMapKey {
	return i.ToEthChainID().MapKey()
}

// AssetMapToArray converts an AssetConfigMap to a slice of AssetConfig.
func AssetMapToArray(am AssetConfigMap) []AssetConfig {
	var arr []AssetConfig
	for _, a := range am {
		arr = append(arr, a)
	}
	return arr
}

// MakePerunAssets creates a slice of channel.Asset from a slice of Asset and bs.
func MakePerunAssets(in []Asset, bs []int) ([]channel.Asset, error) {
	out := make([]channel.Asset, len(in))
	for i, asset := range in {
		if bs[i] == EthereumIndex {
			ethAsset, ok := asset.(*EthereumAsset)
			if !ok {
				return nil, errors.New("wrong asset type: expected EthereumAsset")
			}
			out[i] = echannel.NewAsset(ethAsset.ChainID.Int, ethAsset.AssetHolder)
		} else {
			solanaAsset, ok := asset.(*SolanaAsset)
			if !ok {
				return nil, errors.New("wrong asset type: expected SolanaAsset")
			}
			mint, err := StringToPublicKey(solanaAsset.Mint)
			if err != nil {
				return nil, err
			}
			out[i] = schannel.NewSolanaCrossAssetFromMint(mint)
		}
	}
	return out, nil
}
