package message

import (
	"errors"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gagliardetto/solana-go"
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

// SendSolTx is a request to send a Solana transaction.
func (c *Connection) SendSolTx(tx *solana.Transaction) (_tx *solana.Transaction, err error) {
	txString := tx.MustToBase64()

	req := &SendSolTx{txString}
	resp, err := c.Request(req)
	if err != nil {
		return nil, err
	}

	_resp, ok := resp.(*SendSolTxResponse)
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

	transaction, err := solana.TransactionFromBase64(_resp.Tx)
	if err != nil {
		log.Println("Error: invalid base64: ", err)
		return nil, err
	}

	return transaction, err
}
