package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	ethchannel "github.com/perun-network/perun-eth-backend/channel"
	swallet "github.com/perun-network/perun-eth-backend/wallet/simple"
)

const (
	txFinalityDepth = 1 // Number of blocks required to confirm a transaction.
)

// DeployContracts deploys the Perun smart contracts on the specified ledger.
func DeployContracts(nodeURL string, chainID uint64, privateKey string) (adj, ah common.Address) {
	k, err := crypto.HexToECDSA(privateKey[2:]) // remove 0x prefix
	if err != nil {
		panic(err)
	}
	w := swallet.NewWallet(k)
	cb, err := CreateContractBackend(nodeURL, chainID, w)
	if err != nil {
		panic(err)
	}
	acc := accounts.Account{Address: crypto.PubkeyToAddress(k.PublicKey)}

	// Deploy adjudicator.
	adj, err = ethchannel.DeployAdjudicator(context.TODO(), cb, acc)
	if err != nil {
		panic(err)
	}

	// Deploy asset holder.
	ah, err = ethchannel.DeployETHAssetholder(context.TODO(), cb, adj, acc)
	if err != nil {
		panic(err)
	}

	return adj, ah
}

// CreateContractBackend creates a new contract backend.
func CreateContractBackend(
	nodeURL string,
	chainID uint64,
	w *swallet.Wallet,
) (ethchannel.ContractBackend, error) {
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(chainID))
	transactor := swallet.NewTransactor(w, signer)

	ethClient, err := ethclient.Dial(nodeURL)
	if err != nil {
		return ethchannel.ContractBackend{}, err
	}

	return ethchannel.NewContractBackend(ethClient, ethchannel.MakeChainID(big.NewInt(int64(chainID))), transactor, txFinalityDepth), nil
}

func WriteFrontendConfig(filepath string, ethAH common.Address) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write the frontend configuration to the file.
	_, err = f.WriteString(fmt.Sprintf(ethAH.Hex()))
	return err
}
