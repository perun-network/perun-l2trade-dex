package wallet

import (
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/perun-network/perun-dex-websocket/internal/message"
)

// TransactorFactory is a factory for creating transaction authenticators.
type TransactorFactory struct {
	mtx    sync.RWMutex
	conn   *message.Connection
	sender common.Address
	signer types.Signer
}

// NewTransactorFactory creates a new transaction authenticator factory.
func NewTransactorFactory(
	conn *message.Connection,
	sender common.Address,
	signer types.Signer,
) *TransactorFactory {
	return &TransactorFactory{
		mtx:    sync.RWMutex{},
		conn:   conn,
		sender: sender,
		signer: signer,
	}
}

// SetSender sets the sender account for the transaction authenticator.
func (f *TransactorFactory) SetSender(conn *message.Connection, acc common.Address) {
	f.mtx.Lock()
	f.conn = conn
	f.sender = acc
	f.mtx.Unlock()
}

// NewTransactor creates a new transaction authenticator.
// The given account is ignored in favor of the designated sender.
func (f *TransactorFactory) NewTransactor(_ accounts.Account) (*bind.TransactOpts, error) {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return &bind.TransactOpts{
		From: f.sender,
		Signer: func(addr common.Address, tx *types.Transaction) (_tx *types.Transaction, err error) { // Send Transactions
			f.mtx.RLock()
			defer f.mtx.RUnlock()

			if addr != f.sender {
				err = bind.ErrNotAuthorized
				return
			}

			_tx, err = f.conn.SendETHTx(tx, message.MakeChainID(f.signer.ChainID()))
			if err != nil {
				_tx, err = f.conn.SendETHTx(tx, message.MakeChainID(f.signer.ChainID()))
				if err != nil {
					return nil, err
				}
			}

			// For DynamicFeeTxType transactions, the v value needs to be
			// adapted.
			if _tx.Type() == types.DynamicFeeTxType {
				// Recreate the signature using the r, s, and v values contained in
				// the transaction.
				V, R, S := _tx.RawSignatureValues()
				r, s := R.Bytes(), S.Bytes()
				sig := make([]byte, crypto.SignatureLength)
				copy(sig[32-len(r):32], r)
				copy(sig[64-len(s):64], s)
				sig[64] = byte(V.Uint64())

				// Set the signature correctly using the according signer.
				_tx, err = _tx.WithSignature(f.signer, sig)
				if err != nil {
					return nil, err
				}
			}
			return
		},
	}, nil
}
