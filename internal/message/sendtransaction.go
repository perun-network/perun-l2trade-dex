package message

import (
	"errors"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stellar/go/txnbuild"
)

// SendETHTx is a request to send an Ethereum transaction.
func (c *Connection) SendETHTx(tx *types.Transaction, chainID ChainID) (_tx *types.Transaction, err error) {
	req := &SendETHTx{tx, chainID}
	resp, err := c.Request(req)
	if err != nil {
		return
	}

	_resp, ok := resp.(*SendETHTxResponse)
	if !ok {
		if errMsg, ok := resp.(*Error); ok {
			err = fmt.Errorf("sendTx: %v", errMsg.Err)
		} else {
			err = fmt.Errorf("expected send transaction response, got %T", resp)
		}
		return
	}
	if _resp.Tx == nil {
		err = errors.New("client rejected sending tx")
		return
	}
	_tx = _resp.Tx
	return
}

// SendSteTx is a request to send a Stellar transaction.
func (c *Connection) SendSteTx(xdrString string) (_tx *txnbuild.Transaction, err error) {
	req := &SendSteTx{xdrString}
	resp, err := c.Request(req)
	if err != nil {
		return nil, err
	}

	_resp, ok := resp.(*SendSteTxResponse)
	if !ok {
		if errMsg, ok := resp.(*Error); ok {
			err = fmt.Errorf("sendTx: %v", errMsg.Err)
		} else {
			err = fmt.Errorf("expected send transaction response, got %T", resp)
		}
		return nil, err
	}

	if _resp.Tx == "" {
		err = errors.New("client rejected sending tx")
		return nil, err
	}

	transaction, err := txnbuild.TransactionFromXDR(_resp.Tx)
	if err != nil {
		log.Println("Error: invalid xdr: ", err)
		return nil, err
	}

	tx, ok := transaction.Transaction()
	if !ok {
		log.Println("Error: could not parse transaction from string")
		return nil, errors.New("Could not parse transaction from string")
	}
	_tx = tx
	return _tx, err
}
