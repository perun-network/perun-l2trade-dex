package wallet

import (
	"github.com/perun-network/perun-dex-websocket/internal/message"
	"perun.network/perun-stellar-backend/wallet/types"

	ethwallet "github.com/perun-network/perun-eth-backend/wallet"
	"github.com/pkg/errors"
	"perun.network/go-perun/wallet"
)

// SolWallet is a wrapper around a Solana wallet.
type SolWallet struct {
	*message.Connection
}

// EthWallet is a wrapper around an Ethereum wallet.
type EthWallet struct {
	*message.Connection
}

// Wallet is a wrapper around a wallet.
type Wallet struct {
	*message.Connection
}

// NewWallet creates a new wallet.
func NewWallet(conn *message.Connection) *Wallet {
	return &Wallet{conn}
}

// Unlock unlocks the given address.
func (w *Wallet) Unlock(addr wallet.Address) (acc wallet.Account, err error) {
	_addr, ok := addr.(*ethwallet.Address)
	if !ok {
		err = errors.New("address has invalid type")
		return
	}
	acc = &Account{addr: _addr, w: w}
	return
}

// LockAll locks all accounts.
func (w *Wallet) LockAll() {}

// IncrementUsage increments the usage of the given address.
func (w *Wallet) IncrementUsage(wallet.Address) {}

// DecrementUsage decrements the usage of the given address.
func (w *Wallet) DecrementUsage(wallet.Address) {}

// NewSolWallet creates a new Solana wallet.
func NewSolWallet(conn *message.Connection) *SolWallet {
	return &SolWallet{conn}
}

// Unlock unlocks the given address.
func (w *SolWallet) Unlock(addr wallet.Address) (acc wallet.Account, err error) {
	_addr, ok := addr.(*types.Participant)
	if !ok {
		err = errors.New("address has invalid type")
		return
	}
	_acc := GetAccount(_addr)
	if _acc == nil {
		return nil, errors.New("account not found")
	}
	acc = _acc
	return
}

// LockAll locks all accounts.
func (w *SolWallet) LockAll() {}

// IncrementUsage increments the usage of the given address.
func (w *SolWallet) IncrementUsage(wallet.Address) {}

// DecrementUsage decrements the usage of the given address.
func (w *SolWallet) DecrementUsage(wallet.Address) {}

// NewEthWallet creates a new Ethereum wallet.
func NewEthWallet(conn *message.Connection) *EthWallet {
	return &EthWallet{conn}
}

// Unlock unlocks the given address.
func (w *EthWallet) Unlock(addr wallet.Address) (acc wallet.Account, err error) {
	_addr, ok := addr.(*ethwallet.Address)
	if !ok {
		err = errors.New("address has invalid type")
		return
	}
	_acc := GetAccount(_addr)
	if _acc == nil {
		return nil, errors.New("account not found")
	}
	acc = _acc
	return
}

// LockAll locks all accounts.
func (w *EthWallet) LockAll() {}

// IncrementUsage increments the usage of the given address.
func (w *EthWallet) IncrementUsage(wallet.Address) {}

// DecrementUsage decrements the usage of the given address.
func (w *EthWallet) DecrementUsage(wallet.Address) {}
