package websocket

import (
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/spf13/viper"
	"perun.network/perun-stellar-backend/channel/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mitchellh/mapstructure"

	"github.com/perun-network/perun-dex-websocket/internal/client"
	"github.com/perun-network/perun-dex-websocket/internal/message"

	"github.com/pkg/errors"
)

type (
	// DeployEthereumConfig contains the configuration for deploying contracts on Ethereum.
	DeployEthereumConfig struct {
		ChainsInput EthereumChainsConfig
		// ChainsOutput is the name of the file where the chains config
		// should be written to, once all contracts are deployed.
		ChainsOutput    string
		DialTimeOut     time.Duration
		DeployTimeout   time.Duration
		TxFinalityDepth uint64
	}

	// DeploySolanaConfig contains the configuration for deploying contracts on Solana.
	DeploySolanaConfig struct {
		ChainsInput SolanaChainsConfig
		// ChainsOutput is the name of the file where the chains config
		// should be written to, once all contracts are deployed.
		ChainsOutput    string
		DialTimeOut     time.Duration
		DeployTimeout   time.Duration
		TxFinalityDepth uint64
	}

	// Config represents the parsed config file.
	Config struct {
		WSAddress      string
		TLSCertificate string
		TLSPrivKey     string
		ClientConfig   client.Config
	}

	// EthereumChainsConfig represents the parsed chains' config file.
	EthereumChainsConfig struct {
		Chains []EthereumChainConfig `json:"chains"`
	}

	// SolanaChainsConfig represents the parsed chains' config file.
	SolanaChainsConfig struct {
		Chains []SolanaChainConfig `json:"chains"`
	}

	// EthereumChainConfig represents the configuration of an Ethereum chain.
	EthereumChainConfig struct {
		Name        string                        `json:"name"`
		ChainID     message.ChainID               `json:"chainID"`
		NodeURL     string                        `json:"nodeURL"`
		DeployerSK  string                        `json:"deployerSK,omitempty"`
		Adjudicator common.Address                `json:"adjudicator"`
		Assets      []message.EthereumAssetConfig `json:"assets"`
	}
	// SolanaChainConfig represents the configuration of a Solana chain.
	SolanaChainConfig struct {
		Name         string                      `json:"name"`
		ChainID      message.ChainID             `json:"chainID"`
		NodeURL      string                      `json:"nodeURL"`
		Assets       []message.SolanaAssetConfig `json:"assets"`
		PerunAddress string                      `json:"perunAddress"`
	}
)

// ParseEthereumChainsConfig reads the chains' config file and returns it as a ChainsFile
// struct with the chains and deployer secret keys separated.
func ParseEthereumChainsConfig(file string) (EthereumChainsConfig, error) {
	var chainsFile EthereumChainsConfig

	viper.SetConfigFile(file)
	if err := viper.ReadInConfig(); err != nil {
		return chainsFile, err
	}

	opts := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		parseEthereumConfigTypes(),
	))
	if err := viper.Unmarshal(&chainsFile, opts); err != nil {
		return chainsFile, err
	}

	// Is used for checking for duplicate asset codes.
	assetCodes := make(map[message.EthereumAssetCode]bool)

	// Apply the chain's ID to each asset of this chain and check for duplicate
	// asset codes.
	for _, c := range chainsFile.Chains {
		for i, a := range c.Assets {
			if _, ok := assetCodes[message.EthereumAssetCode(a.Code)]; ok {
				return EthereumChainsConfig{}, errors.Errorf("duplicate asset code %v", a.Code)
			}
			c.Assets[i].ChainID = c.ChainID
			assetCodes[message.EthereumAssetCode(a.Code)] = true
		}
	}

	return chainsFile, nil
}

// ParseSolanaChainsConfig reads the chains' config file and returns it as a ChainsFile
// struct with the chains and deployer secret keys separated.
func ParseSolanaChainsConfig(file string) (SolanaChainsConfig, error) {
	log.Println("Reading config file", file)
	var chainsFile SolanaChainsConfig

	fmt.Println("Reading config file")
	viper.SetConfigFile(file)
	if err := viper.ReadInConfig(); err != nil {
		return chainsFile, err
	}

	fmt.Println("Decoding config file")
	opts := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		parseSolanaConfigTypes(),
	))
	fmt.Println(&opts)
	fmt.Println("Unmarshalling config file", chainsFile)
	if err := viper.Unmarshal(&chainsFile, opts); err != nil {
		return chainsFile, err
	}

	// Is used for checking for duplicate asset codes.
	assetCodes := make(map[types.ContractLID]bool)

	fmt.Println("Checking for duplicate asset codes", chainsFile)
	// Apply the chain's ID to each asset of this chain and check for duplicate
	// asset codes.
	for _, c := range chainsFile.Chains {
		for _, a := range c.Assets {
			stea := a
			if _, ok := assetCodes[types.MakeContractID(stea.Code)]; ok {
				log.Println("Duplicate asset code")
				return SolanaChainsConfig{}, errors.Errorf("duplicate asset code %v", stea.Code)
			}
			assetCodes[types.MakeContractID(stea.Code)] = true
		}
	}
	log.Println("Returning chains file")

	return chainsFile, nil
}

// ChainMap returns the chains as a map where the chain's ID is the key.
func (c EthereumChainsConfig) ChainMap() client.EthereumChainMap {
	chains := make(client.EthereumChainMap)
	for _, chain := range c.Chains {
		// The contracts assets are organized as arrays, so we convert them into
		// an asset map where the asset's code is the key.
		assets := make(message.EthereumAssetConfigMap)
		for _, a := range chain.Assets {
			assets[a.Code] = a
		}

		chains[chain.ChainID.MapKey()] = client.EthereumChain{
			Name:    chain.Name,
			ChainID: chain.ChainID,
			NodeURL: chain.NodeURL,
			Contracts: &client.Contracts{
				Adjudicator: chain.Adjudicator,
				Assets:      assets,
			},
		}
	}
	return chains
}

// AssetMap returns the assets as a map where the asset's code is the key.
func (c SolanaChainsConfig) AssetMap() client.SolanaChainMap {
	chains := make(client.SolanaChainMap)
	for i, chain := range c.Chains {
		assets := make(message.SolanaAssetConfigMap)
		for _, a := range chain.Assets {
			assets[a.Mint] = a
		}
		chains[i] = client.SolanaChain{
			NodeURL:      chain.NodeURL,
			ChainID:      chain.ChainID,
			Name:         chain.Name,
			Assets:       assets,
			PerunAddress: c.Chains[0].PerunAddress,
		}
	}
	return chains
}

// GasLimits returns gas limits for the Adjudicator and Depositors.
// If predefined is true, the predefined gas limits are returned, otherwise the GasLimits are set to 0
func GasLimits(predefined bool) client.GasLimits {
	var gasLimits client.GasLimits
	if predefined {
		gasLimits = client.GasLimits{
			GasLimitETHDepositor:   50000,
			GasLimitERC20Depositor: 500000,
			GasLimitAdjudicator:    1000000,
		}
	} else {
		gasLimits = client.GasLimits{
			GasLimitETHDepositor:   0,
			GasLimitERC20Depositor: 0,
			GasLimitAdjudicator:    0,
		}
	}
	return gasLimits
}

// parseEthereumConfigTypes is used by viper to parse the custom types out of the config file.
func parseEthereumConfigTypes() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		switch to {
		case reflect.TypeOf(common.Address{}):
			addr, ok := data.(string)
			if !ok {
				return nil, errors.New("expected a string for an address")
			}
			if len(addr) != 42 {
				return nil, errors.New("ethereum address must be 42 characters long")
			}
			if !common.IsHexAddress(addr) {
				return nil, errors.New("invalid ethereum address")
			}
			return common.HexToAddress(addr), nil
		case message.ReflectEthereumAssetType:
			assType, ok := data.(string)
			if !ok {
				return nil, errors.New("expected a string for asset type")
			}
			return message.ParseEthereumAssetType(assType)
		case reflect.TypeOf(message.ChainID{}):
			var chID int64
			switch d := data.(type) {
			case int:
				chID = int64(d)
			case string:
				i, err := strconv.Atoi(d)
				if err != nil {
					return nil, errors.Wrap(err, "parsing chain ID")
				}
				chID = int64(i)
			default:
				return nil, errors.New("unsupported type for chain ID")
			}
			return message.MakeChainID(big.NewInt(chID)), nil

		default:
			return data, nil
		}
	}
}

// parseSolanaConfigTypes is used by viper to parse the custom types out of the config file.
func parseSolanaConfigTypes() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		switch to {
		case message.ReflectSolanaAssetType:
			fmt.Println("parsing asset type", from, to, data)
			assType, ok := data.(string)
			fmt.Println(assType)
			if !ok {
				return nil, errors.New("expected a string for asset type")
			}
			return message.ParseSolanaAssetType(assType)
		case reflect.TypeOf(message.ChainID{}):
			fmt.Println("parsing chain id", from, to, data)
			var chID int64
			switch d := data.(type) {
			case int:
				chID = int64(d)
			case string:
				i, err := strconv.Atoi(d)
				if err != nil {
					return nil, errors.Wrap(err, "parsing chain ID")
				}
				chID = int64(i)
			default:
				return nil, errors.New("unsupported type for chain ID")
			}
			return message.MakeChainID(big.NewInt(chID)), nil
		case reflect.TypeOf(message.Mint("")):
			mint, ok := data.(string)
			if !ok {
				return nil, errors.New("expected a string for mint address")
			}
			// Accept empty string as valid (for native SOL)
			if mint != "" {
				if _, err := solana.PublicKeyFromBase58(mint); err != nil {
					return nil, errors.Wrap(err, "invalid base58 mint address")
				}
			}
			return message.MakeMint(mint), nil
		default:
			return data, nil
		}
	}
}
