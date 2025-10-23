package wallet

import (
	"crypto/ecdsa"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	ethwallet "github.com/perun-network/perun-eth-backend/wallet"
	solwallet "github.com/perun-network/perun-solana-backend/wallet"
	"github.com/pkg/errors"
	"perun.network/go-perun/wallet"
)

var (
	perunAccounts = make(map[string]wallet.Account)
	accMutex      = sync.Mutex{}
)

// Account is a wrapper around an Ethereum account.
type Account struct {
	addr *ethwallet.Address
	w    *Wallet
}

// Address returns the address of the account.
func (a *Account) Address() wallet.Address {
	return a.addr
}

// SignData signs the given data.
func (a *Account) SignData(data []byte) (sig []byte, err error) {
	sig, err = a.w.SignData(common.Address(*a.addr), data)
	if err != nil {
		return
	}
	return
}

type EthAccount struct {
	addr  *ethwallet.Address
	w     *EthWallet
	l2Key *ecdsa.PrivateKey
}

type SolAccount struct {
	addr *solwallet.Account
	w    *SolWallet
}

func NewSolAccount(addr *solwallet.Account, w *SolWallet) wallet.Account {
	accMutex.Lock()
	defer accMutex.Unlock()
	if perunAccounts == nil {
		perunAccounts = make(map[string]wallet.Account)
	}
	perunAccounts[addr.Address().String()] = SolAccount{addr: addr, w: w}
	return SolAccount{addr: addr, w: w}
}

func (a SolAccount) Address() wallet.Address {
	return a.addr.Address()
}

func (a SolAccount) SignData(data []byte) (sig []byte, err error) {
	return a.addr.SignData(data)
}

func GetParticipant(addr wallet.Address) *solwallet.Participant {
	return solwallet.AsParticipant(addr)
}

func GetAccount(addr wallet.Address) wallet.Account {
	accMutex.Lock()
	defer accMutex.Unlock()
	if _, exists := perunAccounts[addr.String()]; !exists {
		_ = fmt.Errorf("account not found: %v", addr)
		return nil
	}
	return perunAccounts[addr.String()]
}

func NewEthAccount(addr *ethwallet.Address, w *EthWallet, l2Key *ecdsa.PrivateKey) wallet.Account {
	accMutex.Lock()
	defer accMutex.Unlock()
	if perunAccounts == nil {
		perunAccounts = make(map[string]wallet.Account)
	}
	perunAccounts[addr.String()] = &EthAccount{addr: addr, w: w, l2Key: l2Key}
	return &EthAccount{addr: addr, w: w, l2Key: l2Key}
}

func (a *EthAccount) Address() wallet.Address {
	return a.addr
}

func (a *EthAccount) SignData(data []byte) (sig []byte, err error) {
	hash := ethwallet.PrefixedHash(data)
	sig, err = a.SignHash(hash)
	if err != nil {
		return nil, errors.Wrap(err, "SignHash")
	}
	sig[64] += 27
	return
}

// SignHash is used to sign an already prefixed hash with this account.
func (a *EthAccount) SignHash(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, a.l2Key)
}
