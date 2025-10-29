package message

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// SignData is a request to sign data.
func (c *Connection) SignData(addr common.Address, data []byte) (sig []byte, err error) {
	req := &SignETHData{addr, data}
	resp, err := c.Request(req)
	if err != nil {
		return
	}

	_resp, ok := resp.(*SignResponse)
	if !ok {
		err = fmt.Errorf("expected sign data response, got %T", resp)
		return
	}

	sig = _resp.Signature
	return
}

// SignETHData is a request to sign Ethereum data.
func (c *Connection) SignETHData(addr common.Address, data []byte) (sig []byte, err error) {
	req := &SignETHData{addr, data}
	resp, err := c.Request(req)
	if err != nil {
		return
	}

	_resp, ok := resp.(*SignResponse)
	if !ok {
		err = fmt.Errorf("expected sign data response, got %T", resp)
		return
	}

	sig = _resp.Signature
	return
}

// SignSolData is a request to sign Solana data.
func (c *Connection) SignSolData(addr string, data []byte) (sig []byte, err error) {
	req := &SignSolData{addr, data}
	resp, err := c.Request(req)
	if err != nil {
		return
	}

	_resp, ok := resp.(*SignResponse)
	if !ok {
		err = fmt.Errorf("expected sign data response, got %T", resp)
		return
	}

	sig = _resp.Signature
	return
}
