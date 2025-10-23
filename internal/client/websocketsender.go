package client

import (
	"context"
	"log"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"

	"github.com/gagliardetto/solana-go"

	"github.com/perun-network/perun-solana-backend/client"

	"github.com/perun-network/perun-dex-websocket/internal/message"

	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
)

// WebSocketSender implements the sender interface of the solana backend to sign and send solana transactions.
type WebSocketSender struct {
	conn      *message.Connection
	add       *solana.PublicKey
	rpcClient *rpc.Client
}

// NewWebSocketSender creates a new websocket sender.
func NewWebSocketSender(conn *message.Connection, add *solana.PublicKey) client.Sender {
	return &WebSocketSender{conn: conn, add: add}
}

// SignSendTx signs and sends the transaction.
func (s *WebSocketSender) SignSendTx(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {

	_tx, err := s.conn.SendSolTx(tx)
	if err != nil {
		log.Println("Error sending transaction:", err)
		_tx, err = s.conn.SendSolTx(tx)
		if err != nil {
			return solana.Signature{}, err
		}
	}

	sig, err := s.rpcClient.SendTransaction(
		ctx,
		_tx,
	)

	return sig, nil
}

func (s *WebSocketSender) SignSendAndConfirmTx(ctx context.Context, tx *solana.Transaction, wsClient *ws.Client) (solana.Signature, error) {
	_tx, err := s.conn.SendSolTx(tx)
	if err != nil {
		log.Println("Error sending transaction:", err)
		_tx, err = s.conn.SendSolTx(tx)
		if err != nil {
			return solana.Signature{}, err
		}
	}

	sig, err := confirm.SendAndConfirmTransaction(
		ctx,
		s.rpcClient,
		wsClient,
		_tx,
	)

	return sig, nil
}

// SetRPCClient sets the RPC client.
func (s *WebSocketSender) SetRPCClient(rpcClient *rpc.Client) error {
	s.rpcClient = rpcClient
	return nil
}

func (s *WebSocketSender) GetRPCClient() *rpc.Client {
	return s.rpcClient
}
