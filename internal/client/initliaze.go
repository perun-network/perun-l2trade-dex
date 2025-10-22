package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/perun-network/perun-dex-websocket/internal/message"
	wwallet "github.com/perun-network/perun-dex-websocket/internal/wallet"
	ethchannel "github.com/perun-network/perun-eth-backend/channel"
	ethwallet "github.com/perun-network/perun-eth-backend/wallet"
	ethwire "github.com/perun-network/perun-eth-backend/wire"
	solchannel "github.com/perun-network/perun-solana-backend/channel"
	soladj "github.com/perun-network/perun-solana-backend/channel/adjudicator"
	solfunder "github.com/perun-network/perun-solana-backend/channel/funder"
	solclient "github.com/perun-network/perun-solana-backend/client"
	solwallet "github.com/perun-network/perun-solana-backend/wallet"
	"perun.network/go-perun/channel/multi"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/watcher/local"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/net/simple"
)

// NodeURL is the URL of an Ethereum node.
type NodeURL = string

var (
	bus           = wire.NewLocalBus()
	ethClients    = make(map[NodeURL]*ethclient.Client)
	ethClientsMtx = sync.Mutex{}
)

// WrappedContractInterface is a wrapper over the contract backend which
// implements SendTransaction as a no-op.
type WrappedContractInterface struct {
	ethchannel.ContractInterface
}

// SendTransaction is a no-op since the transaction is sent by the websocket
// client.
func (b WrappedContractInterface) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return nil
}

func newPerunClient(
	conn *message.Connection,
	l2sk *ecdsa.PrivateKey,
	eaddr common.Address,
	saddr string,
	cfg Config,
) (
	map[wallet.BackendID]wallet.Address,
	*client.Client,
	*multi.Adjudicator,
	transactionFactoryMap,
	map[wallet.BackendID]wire.Address,
	error,
) {
	log.Println("Initializing Perun client", eaddr, saddr)
	ethWall := wwallet.NewEthWallet(conn)
	// Prepare Solana contract backend
	fromAddress, err := solana.PublicKeyFromBase58(saddr)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Convert the private key to bytes
	privateKeyBytes := crypto.FromECDSA(l2sk)

	// Convert the private key to a hex string
	privateKeyHex := fmt.Sprintf("%x", privateKeyBytes)
	l2Address := crypto.PubkeyToAddress(l2sk.PublicKey)
	steWall := wwallet.NewSolWallet(conn)
	account, err := solwallet.NewAccount(privateKeyHex, fromAddress, l2Address)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	part := account.Participant()
	solAcc := wwallet.NewSolAccount(account, steWall)
	sWall := solwallet.NewEphemeralWallet()
	err = sWall.AddAccount(account)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	sender := NewWebSocketSender(conn, &fromAddress)
	tc := solclient.NewSignerConfig(
		nil,
		nil,
		solAcc,
		sender,
		rpc.LocalNet_RPC, // Use the localnet RPC URL
	)
	cb := solclient.NewContractBackend(*tc, solchannel.BackendID)

	// Prepare Ethereum contract backend
	_ = wwallet.NewEthAccount(ethwallet.AsWalletAddr(l2Address), ethWall, l2sk)

	// Prepare multi Funder, Watcher & Adjudicator
	multiFunder := multi.NewFunder()
	multiAdjudicator := multi.NewAdjudicator()
	watcher, err := local.NewWatcher(multiAdjudicator)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("initializing watcher: %w", err)
	}

	// Get all assets of all chains.
	assets := cfg.EthChains.Assets()
	adjTfs := make(transactionFactoryMap)
	adjs := make(map[multi.LedgerIDMapKey]*soladj.Adjudicator)

	// Register all solana assets on the funder and add adjudicators
	for _, a := range cfg.SolChains {
		perunAddr, err := message.StringToSolanaPublicKey(a.PerunAddress)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		vec, err := message.ConvertAssetConfigMapToVec(a.Assets)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		funder := solfunder.NewFunder(cb, perunAddr, vec)
		multiFunder.RegisterFunder(solchannel.MakeCCID(solchannel.MakeContractID("6")), funder)

		adjudicator := soladj.NewAdjudicator(cb, perunAddr, vec, false)
		multiAdjudicator.RegisterAdjudicator(solchannel.MakeCCID(solchannel.MakeContractID("6")), adjudicator)

		adjs[solchannel.MakeContractID("6").MapKey()] = adjudicator
	}

	// Register all ethereum assets on the funder and add adjudicators.
	for id, c := range cfg.EthChains {
		ethClient, err := getEthClient(c.NodeURL)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		ci := &WrappedContractInterface{ethClient}
		signer := types.NewLondonSigner(c.ChainID.Int)

		tfAh := wwallet.NewTransactorFactory(conn, eaddr, signer)

		cbAh := ethchannel.NewContractBackend(ci, c.ChainID.ToEthChainID(), tfAh, cfg.TxFinalityDepth)
		funder := ethchannel.NewFunder(cbAh)
		multiFunder.RegisterFunder(ethchannel.MakeLedgerBackendID(c.ChainID.ToEthChainID().Int), funder)
		err = registerAssets(accounts.Account{Address: eaddr}, funder, assets, cfg.GasLimits)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		tfAdj := wwallet.NewTransactorFactory(conn, eaddr, signer)
		cbAdj := ethchannel.NewContractBackend(ci, c.ChainID.ToEthChainID(), tfAdj, cfg.TxFinalityDepth)
		adjudicator := ethchannel.NewAdjudicator(cbAdj, c.Adjudicator, eaddr, accounts.Account{Address: eaddr}, cfg.GasLimits.GasLimitAdjudicator)
		multiAdjudicator.RegisterAdjudicator(ethchannel.MakeLedgerBackendID(c.ChainID.ToEthChainID().Int), adjudicator)

		adjTfs[id] = tfAdj
	}
	walletAddr := map[wallet.BackendID]wallet.Address{message.EthereumIndex: ethwallet.AsWalletAddr(l2Address), message.SolanaIndex: part}
	wireAddr := map[wallet.BackendID]wire.Address{message.EthereumIndex: &ethwire.Address{Address: ethwallet.AsWalletAddr(l2Address)}, message.SolanaIndex: simple.NewAddress(part.String())}
	perunClient, err := client.New(wireAddr, bus, multiFunder,
		multiAdjudicator, map[wallet.BackendID]wallet.Wallet{message.EthereumIndex: ethWall, message.SolanaIndex: sWall}, watcher)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return walletAddr, perunClient, multiAdjudicator, adjTfs, wireAddr, nil
}

// registerAssets registers the given `assets` on the funder.
func registerAssets(acc accounts.Account, funder *ethchannel.Funder, assets []message.EthereumAssetConfig, gasLimits GasLimits) error {
	for _, a := range assets {
		eAsset := *ethchannel.NewAsset(a.ChainID.Int, a.AssetHolder)
		if _, _, ok := funder.IsAssetRegistered(eAsset); ok {
			continue
		}

		var depositor ethchannel.Depositor
		if a.ChainID.MapKey() != funder.ChainID().MapKey() {
			depositor = ethchannel.NewNoOpDepositor()
		} else {
			switch a.Type {
			case message.AssetTypeETH:
				depositor = ethchannel.NewETHDepositor(gasLimits.GasLimitETHDepositor)
			case message.AssetTypeERC20:
				depositor = ethchannel.NewERC20Depositor(a.Address, gasLimits.GasLimitERC20Depositor)
			}
		}

		ok := funder.RegisterAsset(eAsset, depositor, acc)
		if !ok {
			return fmt.Errorf("could not register asset %v", a)
		}
	}
	return nil
}

// getEthClient returns an Ethereum client for the given `nodeURL`.
// It first tries to get an already existent Ethereum client, if there exists no
// client for this nodeURL yet, we create a new one and return it.
func getEthClient(nodeURL string) (*ethclient.Client, error) {
	ethClientsMtx.Lock()
	defer ethClientsMtx.Unlock()
	ethClient, ok := ethClients[nodeURL]
	if !ok {
		var err error
		ethClient, err = ethclient.Dial(nodeURL)
		if err != nil {
			return nil, err
		}
		ethClients[nodeURL] = ethClient
	}

	return ethClient, nil
}
