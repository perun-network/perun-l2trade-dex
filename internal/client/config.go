package client

import (
	"time"

	"github.com/ethereum/go-ethereum/common"

	"perun.network/go-perun/channel/multi"

	"github.com/perun-network/perun-dex-websocket/internal/message"
)

type (
	// Config contains the configuration for the client.
	Config struct {
		Timeouts
		TxFinalityDepth uint64
		EthChains       EthereumChainMap
		SolChains       SolanaChainMap
		GasLimits       GasLimits
		HorizonURL      string
	}

	// Timeouts contains the timeouts for the client.
	Timeouts struct {
		DefaultTimeout time.Duration
		HandleTimeout  time.Duration
		FundTimeout    time.Duration
		SettleTimeout  time.Duration
	}

	// GasLimits contains the gas limits for the client.
	GasLimits struct {
		GasLimitETHDepositor   uint64
		GasLimitERC20Depositor uint64
		GasLimitAdjudicator    uint64
	}

	// EthereumChainMap is a map of Ethereum chains.
	EthereumChainMap map[multi.LedgerIDMapKey]EthereumChain

	// SolanaChainMap is a map of Solana chains.
	SolanaChainMap map[int]SolanaChain

	// EthereumChain contains the configuration for an Ethereum chain.
	EthereumChain struct {
		Name    string
		ChainID message.ChainID
		NodeURL string
		*Contracts
	}

	// SolanaChain contains the configuration for a Solana chain.
	SolanaChain struct {
		NodeURL      string
		Name         string
		ChainID      message.ChainID
		Assets       message.SolanaAssetConfigMap
		PerunAddress string
	}

	// Contracts contains the addresses of the contracts on the chain.
	Contracts struct {
		Adjudicator common.Address
		Assets      message.EthereumAssetConfigMap
	}
)

// Assets returns all assets in the Ethereum chain map.
func (c EthereumChainMap) Assets() []message.EthereumAssetConfig {
	var assets []message.EthereumAssetConfig
	for _, chain := range c {
		for _, a := range chain.Assets {
			assets = append(assets, a)
		}
	}
	return assets
}
