package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/stellar/go/keypair"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
)

type (
	// Message is the interface for all messages that can be sent over the
	Message interface{ messageType() string }
	// JSONObject is a wrapper for a Message that includes the message type.
	JSONObject struct {
		Message
	}

	// Request is the wrapper for a message that is sent by the WebSocket client.
	Request struct {
		ID      uint64      `json:"id"`
		Message *JSONObject `json:"message"`
	}

	// Response is the wrapper for a message that is sent by the Perun client.
	Response struct {
		ID      uint64      `json:"id"`
		Message *JSONObject `json:"message"`
	}

	// BlockchainClient interface defines the common behavior for different blockchain clients.
	BlockchainClient interface {
		ParseClientAddress() (interface{}, error)
	}

	// EthereumInitialize represents the initialization message for Ethereum blockchain.
	EthereumInitialize struct {
		ClientAddress  common.Address `json:"clientAddress"`
		EgoisticClient bool           `json:"egoisticClient"`
	}

	// CrossContractInitialize represents the initialization message for crossChain blockchain.
	CrossContractInitialize struct {
		EthClientAddress common.Address `json:"ethClientAddress"`
		SolClientAddress string         `json:"solClientAddress"`
		EgoisticClient   bool           `json:"egoisticClient"`
	}

	// SolanaInitialize represents the initialization message for Solana blockchain.
	SolanaInitialize struct {
		ClientAddress  string `json:"clientAddress"`
		EgoisticClient bool   `json:"egoisticClient"`
	}

	// Initialize is the initial message expected to be sent by the WebSocket
	// client and runs the Perun client for the given address.
	//
	// This is the only message sent by the WebSocket client that is not wrapped
	// in a Request or Response.

	// Initialized is sent to the WebSocket client after the Perun client has
	// been successfully started.
	Initialized struct {
		L2Address common.Address `json:"l2Address"`
	}

	// SetAdjTxSender can be used to set the adjudicator transaction sender for
	// a given chain.
	SetAdjTxSender struct {
		ChainID ChainID        `json:"chainID"`
		Sender  common.Address `json:"sender"`
	}

	// ChainInfo is the representation of a chain.
	ChainInfo struct {
		Name string  `json:"name"`
		ID   ChainID `json:"ID"`
	}

	// GetChains is requested by the WebSocket clients to query for the
	// supported chains.
	GetChains struct{}

	// GetChainsResponse is the response to a GetChains request to inform about
	// the supported chains represented by their name and ID.
	GetChainsResponse struct {
		Chains []ChainInfo `json:"chains"`
	}

	// GetAssets is sent by the WebSocket client to query for the supported
	// assets on the chains with the matching ID.
	GetAssets struct {
		ChainIDs []ChainID `json:"chainIDs"`
	}

	// GetAssetsResponse is the response to a GetAssets request and includes the
	// supported solana assets.
	GetAssetsResponse struct {
		SolanaAssets   []SolanaAssetConfig   `json:"solanaAssets"`
		EthereumAssets []EthereumAssetConfig `json:"ethereumAssets"`
	}

	// GetDecimals is sent by the WebSocket client to query for the precision
	// of the asset.
	GetDecimals struct {
		Asset Asset `json:"Asset"`
	}

	// GetDecimalsResponse is the response to a GetDecimals request and includes
	// the decimals for the requested asset.
	GetDecimalsResponse struct {
		Decimals uint8 `json:"decimals"`
	}

	// GetTimeout is sent by the WebSocket client to query the duration of a
	// specific timeout.
	GetTimeout struct {
		TimeoutType TimeoutType `json:"timeoutType"`
	}

	// GetTimeoutResponse is the response to a GetTimeout request and includes
	// the duration of the requested timeout in seconds.
	GetTimeoutResponse struct {
		Timeout int64 `json:"timeout"`
	}

	// GetQuote is sent by the WebSocket client to query the quote from
	// `FromAsset` to `ToAsset` from the hub. It is then forwarded to the hub.
	GetQuote struct {
		FromAsset Asset  `json:"fromAsset"`
		ToAsset   Asset  `json:"toAsset"`
		Hub       string `json:"hub"`
	}

	// GetQuoteResponse is sent by the hub as a response to a GetQuote request
	// and is forwarded to the requesting WebSocket client.
	GetQuoteResponse struct {
		CrossQuote float64 `json:"crossQuote"`
		FromQuote  float64 `json:"fromQuote"`
		ToQuote    float64 `json:"toQuote"`
		FromGas    string  `json:"fromGas"`
		ToGas      string  `json:"toGas"`
	}

	// GetBalance allows the user to get the balance from the contractBackend.
	GetBalance struct {
		Asset SolanaAsset `json:"asset"`
	}

	// GetHubBalance allows the user to get the hub's balance from the contractBackend.
	GetHubBalance struct {
		Asset Asset  `json:"asset"`
		Hub   string `json:"hub"`
	}

	// GetBalanceResponse returns the balance for the given asset.
	GetBalanceResponse struct {
		Balance string `json:"balance"`
	}

	// GetFunds is sent by the hub to request minting funds.
	GetFunds struct {
		Address string `json:"address"`
		Asset   string `json:"asset"`
		Amount  string `json:"amount"`
	}

	// GetFundsResponse is the response to a GetFunds request.
	GetFundsResponse struct {
		Success bool `json:"success"`
	}

	// OpenChannel is sent by the WebSocket client to make a channel proposal to
	// PeerAddress. The proposalID will be included in the corresponding
	// ChannelCreated message if the channel has been created.
	OpenChannel struct {
		ProposalID        client.ProposalID `json:"proposalID"`
		PeerAddressEth    common.Address    `json:"peerAddressEth"`
		PeerAddressSol    string            `json:"peerAddressSol"`
		ChallengeDuration uint64            `json:"challengeDuration,string"`
		State             ChannelState      `json:"state"`
	}

	// UpdateChannel is used to propose a channel update proposal or to notify
	// about an incoming update proposal.
	UpdateChannel struct {
		ID    channel.ID   `json:"id"`
		State ChannelState `json:"state"`
	}

	// ChannelProposal is used to notify the WebSocket client about an incoming
	// channel proposal.
	ChannelProposal struct {
		ID             client.ProposalID `json:"ID"`
		PeerAddressEth common.Address    `json:"peerAddressEth"`
		PeerAddressSol string            `json:"peerAddressSol"`
		State          ChannelState      `json:"state"`
	}

	// ProposalResponse is used as a response to an OpenChannel or UpdateChannel
	// request.
	ProposalResponse struct {
		Accepted     bool   `json:"accepted"`
		RejectReason string `json:"rejectReason,omitempty"`
	}

	// ChannelCreated is sent to the WebSocket client to notify that a channel
	// has been created which originates from the proposal with the included ID.
	ChannelCreated struct {
		ID         channel.ID        `json:"id"`
		ProposalID client.ProposalID `json:"proposalID"`
		Idx        channel.Index     `json:"idx"`
	}

	// CloseChannel is used by the WebSocket client for closing a channel.
	// If a withdrawal address is provided, the funds of the channel are
	// withdrawn to this address. Default is the address of the client.
	// If the ForceClose flag is true, a dispute is registered to force the
	// settlement.
	CloseChannel struct {
		ID                channel.ID `json:"id"`
		WithdrawalAddress *string    `json:"withdrawalAddress,omitempty"`
		ForceClose        bool       `json:"forceClose"`
	}

	// ChannelClosed is sent to the WebSocket client to notify that a channel
	// has been closed.
	ChannelClosed struct {
		ID channel.ID `json:"id"`
	}

	// GetChannelInfo is used by the WebSocket client to request channel
	// information for the channel with the given ID.
	GetChannelInfo struct {
		ID channel.ID `json:"id"`
	}

	// ChannelInfo is the response to a GetChannelInfo request.
	ChannelInfo struct {
		PeerAddressEth common.Address `json:"peerAddressEth"`
		PeerAddressSol string         `json:"peerAddressSol"`
		State          ChannelState   `json:"state"`
	}

	// GetSignedState is used by the WebSocket client to request the signed
	// state of the channel with the given ID. This allows the client to recover
	// its funds even if the websocket may be offline.
	GetSignedState struct {
		ID channel.ID `json:"id"`
	}

	// SendSignedState is used by the Perun-x-Frontend to send the SignedState back
	SendSignedState SignedState

	// SignETHData is sent to the WebSocket client to request the signing of Data.
	SignETHData struct {
		Address common.Address `json:"address"`
		Data    []byte         `json:"data"`
	}

	// SignSolData is sent to the WebSocket client to request the signing of Data.
	SignSolData struct {
		Address string `json:"address"`
		Data    []byte `json:"data"`
	}

	// SendETHTx is sent to the WebSocket client to request the sending of a
	// transaction.
	// SendETHTx is also sent by the client to response the potentially modified
	// transaction.
	SendETHTx struct {
		Tx      *types.Transaction `json:"transaction"`
		ChainID ChainID            `json:"chainID"`
	}

	// SendETHTxResponse is the response from the client to a SendETHTx request and
	// contains the signed transaction.
	SendETHTxResponse struct {
		Tx *types.Transaction `json:"transaction"`
	}

	// SendSteTx is sent to the WebSocket client to request the sending of a
	// transaction.
	// SendSolTx is also sent by the client to response the potentially modified
	// transaction.
	SendSolTx struct {
		Tx string `json:"transaction"`
	}

	// SendSolTxResponse is the response from the client to a SendSolTx request and
	// contains the signed transaction.
	SendSolTxResponse struct {
		Tx string `json:"transaction"`
	}

	// SignResponse is sent by the WebSocket client to respond to a SignETHData
	// request. It contains the requested Signature.
	SignResponse struct {
		Signature []byte `json:"signature"`
	}

	// Success is sent in a Response to signal success of a Request.
	Success struct{}

	// FundingError is sent to the client if the funding fails for the channel
	// with ChannelID which originated from the proposal with ProposalID.
	FundingError struct {
		ProposalID client.ProposalID `json:"proposalID"`
		ChannelID  channel.ID        `json:"channelID"`
		Err        string            `json:"error"`
	}

	// CreateOrder is sent by the maker to propose a new order to the peer.
	CreateOrder struct {
		Order Order `json:"order"`
	}

	// CreateOrderAck is returned by the taker (or your client) indicating local acceptance
	// of displaying/keeping the order in the off-chain book; trade still requires ch.Update.
	CreateOrderAck struct {
		ID        OrderID `json:"id"`
		Accepted  bool    `json:"accepted"`
		Reason    string  `json:"reason,omitempty"`
		TotalOpen uint64  `json:"totalOpen"`
	}

	// CancelOrder removes an active order from the off-chain book.
	CancelOrder struct {
		ChannelID channel.ID `json:"channelID"`
		ID        OrderID    `json:"id"`
		Reason    string     `json:"reason,omitempty"`
	}

	// CancelOrderAck confirms removal.
	CancelOrderAck struct {
		ID        OrderID `json:"id"`
		Success   bool    `json:"success"`
		Reason    string  `json:"reason,omitempty"`
		TotalOpen uint64  `json:"totalOpen"`
	}

	// AcceptOrder signals the taker wants to accept an order; the maker then proceeds
	// to invoke ch.Update (or the taker, depending on your trading protocol) to settle.
	// This is a control-plane message; the asset movement will be in the channel update.
	AcceptOrder struct {
		ChannelID channel.ID `json:"channelID"`
		ID        OrderID    `json:"id"`
		// Optional: partial fill amount; omit for full fill.
		Amount string `json:"amount,omitempty"`
	}

	// AcceptOrderAck returns the decision prior to any ch.Update.
	AcceptOrderAck struct {
		ID       OrderID `json:"id"`
		Accepted bool    `json:"accepted"`
		Reason   string  `json:"reason,omitempty"`
	}

	// GetOrderBook requests either a snapshot or a delta since the given sequence.
	// If SinceSequence == 0, a snapshot should be returned.
	GetOrderBook struct {
		ChannelID     channel.ID `json:"channelID"`
		SinceSequence uint64     `json:"sinceSequence"`
	}

	// GetOrderBookResponse returns a snapshot or delta; only one of the fields is set.
	GetOrderBookResponse struct {
		Snapshot *OrderBookSnapshot `json:"snapshot,omitempty"`
		Delta    *OrderBookDelta    `json:"delta,omitempty"`
	}

	// Error is sent as a response to notify the client/WebSocket about an error.
	Error struct {
		Err string `json:"error"`
	}

	// MockMessage is only used for testing the correctness of (un)marshalling
	// messages.
	MockMessage struct{}
)

// messageTypes is a map from the message type names to their reflected type.
// It is used to unmarshal messages when having their message type name.
var messageTypes = map[string]reflect.Type{
	(*Request)(nil).messageType():                 reflect.ValueOf((*Request)(nil)).Type().Elem(),
	(*Response)(nil).messageType():                reflect.ValueOf((*Response)(nil)).Type().Elem(),
	(*EthereumInitialize)(nil).messageType():      reflect.ValueOf((*EthereumInitialize)(nil)).Type().Elem(),
	(*SolanaInitialize)(nil).messageType():        reflect.ValueOf((*SolanaInitialize)(nil)).Type().Elem(),
	(*CrossContractInitialize)(nil).messageType(): reflect.ValueOf((*CrossContractInitialize)(nil)).Type().Elem(),
	(*Initialized)(nil).messageType():             reflect.ValueOf((*Initialized)(nil)).Type().Elem(),
	(*SetAdjTxSender)(nil).messageType():          reflect.ValueOf((*SetAdjTxSender)(nil)).Type().Elem(),
	(*GetChains)(nil).messageType():               reflect.ValueOf((*GetChains)(nil)).Type().Elem(),
	(*GetChainsResponse)(nil).messageType():       reflect.ValueOf((*GetChainsResponse)(nil)).Type().Elem(),
	(*GetAssets)(nil).messageType():               reflect.ValueOf((*GetAssets)(nil)).Type().Elem(),
	(*GetAssetsResponse)(nil).messageType():       reflect.ValueOf((*GetAssetsResponse)(nil)).Type().Elem(),
	(*GetDecimals)(nil).messageType():             reflect.ValueOf((*GetDecimals)(nil)).Type().Elem(),
	(*GetDecimalsResponse)(nil).messageType():     reflect.ValueOf((*GetDecimalsResponse)(nil)).Type().Elem(),
	(*GetBalance)(nil).messageType():              reflect.ValueOf((*GetBalance)(nil)).Type().Elem(),
	(*GetHubBalance)(nil).messageType():           reflect.ValueOf((*GetHubBalance)(nil)).Type().Elem(),
	(*GetBalanceResponse)(nil).messageType():      reflect.ValueOf((*GetBalanceResponse)(nil)).Type().Elem(),
	(*GetTimeout)(nil).messageType():              reflect.ValueOf((*GetTimeout)(nil)).Type().Elem(),
	(*GetTimeoutResponse)(nil).messageType():      reflect.ValueOf((*GetTimeoutResponse)(nil)).Type().Elem(),
	(*GetQuote)(nil).messageType():                reflect.ValueOf((*GetQuote)(nil)).Type().Elem(),
	(*GetQuoteResponse)(nil).messageType():        reflect.ValueOf((*GetQuoteResponse)(nil)).Type().Elem(),
	(*GetFunds)(nil).messageType():                reflect.ValueOf((*GetFunds)(nil)).Type().Elem(),
	(*GetFundsResponse)(nil).messageType():        reflect.ValueOf((*GetFundsResponse)(nil)).Type().Elem(),
	(*OpenChannel)(nil).messageType():             reflect.ValueOf((*OpenChannel)(nil)).Type().Elem(),
	(*UpdateChannel)(nil).messageType():           reflect.ValueOf((*UpdateChannel)(nil)).Type().Elem(),
	(*ChannelProposal)(nil).messageType():         reflect.ValueOf((*ChannelProposal)(nil)).Type().Elem(),
	(*ProposalResponse)(nil).messageType():        reflect.ValueOf((*ProposalResponse)(nil)).Type().Elem(),
	(*ChannelCreated)(nil).messageType():          reflect.ValueOf((*ChannelCreated)(nil)).Type().Elem(),
	(*CloseChannel)(nil).messageType():            reflect.ValueOf((*CloseChannel)(nil)).Type().Elem(),
	(*ChannelClosed)(nil).messageType():           reflect.ValueOf((*ChannelClosed)(nil)).Type().Elem(),
	(*GetChannelInfo)(nil).messageType():          reflect.ValueOf((*GetChannelInfo)(nil)).Type().Elem(),
	(*ChannelInfo)(nil).messageType():             reflect.ValueOf((*ChannelInfo)(nil)).Type().Elem(),
	(*GetSignedState)(nil).messageType():          reflect.ValueOf((*GetSignedState)(nil)).Type().Elem(),
	(*SendSignedState)(nil).messageType():         reflect.ValueOf((*SendSignedState)(nil)).Type().Elem(),
	(*SignedState)(nil).messageType():             reflect.ValueOf((*SignedState)(nil)).Type().Elem(),
	(*SignETHData)(nil).messageType():             reflect.ValueOf((*SignETHData)(nil)).Type().Elem(),
	(*SignSolData)(nil).messageType():             reflect.ValueOf((*SignSolData)(nil)).Type().Elem(),
	(*SendETHTx)(nil).messageType():               reflect.ValueOf((*SendETHTx)(nil)).Type().Elem(),
	(*SendETHTxResponse)(nil).messageType():       reflect.ValueOf((*SendETHTxResponse)(nil)).Type().Elem(),
	(*SendSolTx)(nil).messageType():               reflect.ValueOf((*SendSolTx)(nil)).Type().Elem(),
	(*SendSolTxResponse)(nil).messageType():       reflect.ValueOf((*SendSolTxResponse)(nil)).Type().Elem(),
	(*SignResponse)(nil).messageType():            reflect.ValueOf((*SignResponse)(nil)).Type().Elem(),
	(*FundingError)(nil).messageType():            reflect.ValueOf((*FundingError)(nil)).Type().Elem(),
	(*OrderBookSnapshot)(nil).messageType():       reflect.ValueOf((*OrderBookSnapshot)(nil)).Type().Elem(),
	(*OrderBookDelta)(nil).messageType():          reflect.ValueOf((*OrderBookDelta)(nil)).Type().Elem(),
	(*CreateOrder)(nil).messageType():             reflect.ValueOf((*CreateOrder)(nil)).Type().Elem(),
	(*CreateOrderAck)(nil).messageType():          reflect.ValueOf((*CreateOrderAck)(nil)).Type().Elem(),
	(*CancelOrder)(nil).messageType():             reflect.ValueOf((*CancelOrder)(nil)).Type().Elem(),
	(*CancelOrderAck)(nil).messageType():          reflect.ValueOf((*CancelOrderAck)(nil)).Type().Elem(),
	(*AcceptOrder)(nil).messageType():             reflect.ValueOf((*AcceptOrder)(nil)).Type().Elem(),
	(*AcceptOrderAck)(nil).messageType():          reflect.ValueOf((*AcceptOrderAck)(nil)).Type().Elem(),
	(*GetOrderBook)(nil).messageType():            reflect.ValueOf((*GetOrderBook)(nil)).Type().Elem(),
	(*GetOrderBookResponse)(nil).messageType():    reflect.ValueOf((*GetOrderBookResponse)(nil)).Type().Elem(),
	(*Error)(nil).messageType():                   reflect.ValueOf((*Error)(nil)).Type().Elem(),
	(*Success)(nil).messageType():                 reflect.ValueOf((*Success)(nil)).Type().Elem(),
	(*MockMessage)(nil).messageType():             reflect.ValueOf((*MockMessage)(nil)).Type().Elem(),
}

func (*Request) messageType() string                 { return "Request" }
func (*Response) messageType() string                { return "Response" }
func (*EthereumInitialize) messageType() string      { return "EthereumInitialize" }
func (*SolanaInitialize) messageType() string        { return "SolanaInitialize" }
func (*CrossContractInitialize) messageType() string { return "CrossContractInitialize" }
func (*Initialized) messageType() string             { return "Initialized" }
func (*SetAdjTxSender) messageType() string          { return "SetAdjTxSender" }
func (*GetChains) messageType() string               { return "GetChains" }
func (*GetChainsResponse) messageType() string       { return "GetChainsResponse" }
func (*GetAssets) messageType() string               { return "GetAssets" }
func (*GetAssetsResponse) messageType() string       { return "GetAssetsResponse" }
func (*GetDecimals) messageType() string             { return "GetDecimals" }
func (*GetDecimalsResponse) messageType() string     { return "GetDecimalsResponse" }
func (*GetTimeout) messageType() string              { return "GetTimeout" }
func (*GetTimeoutResponse) messageType() string      { return "GetTimeoutResponse" }
func (*GetQuote) messageType() string                { return "GetQuote" }
func (*GetQuoteResponse) messageType() string        { return "GetQuoteResponse" }
func (*GetFunds) messageType() string                { return "GetFunds" }
func (*GetFundsResponse) messageType() string        { return "GetFundsResponse" }
func (*GetBalance) messageType() string              { return "GetBalance" }
func (*GetHubBalance) messageType() string           { return "GetHubBalance" }
func (*GetBalanceResponse) messageType() string      { return "GetBalanceResponse" }
func (*OpenChannel) messageType() string             { return "OpenChannel" }
func (*UpdateChannel) messageType() string           { return "UpdateChannel" }
func (*ChannelProposal) messageType() string         { return "ChannelProposal" }
func (*ProposalResponse) messageType() string        { return "ProposalResponse" }
func (*ChannelCreated) messageType() string          { return "ChannelCreated" }
func (*CloseChannel) messageType() string            { return "CloseChannel" }
func (*ChannelClosed) messageType() string           { return "ChannelClosed" }
func (*GetChannelInfo) messageType() string          { return "GetChannelInfo" }
func (*ChannelInfo) messageType() string             { return "ChannelInfo" }
func (*GetSignedState) messageType() string          { return "GetSignedState" }
func (*SignedState) messageType() string             { return "SignedState" }
func (*SignETHData) messageType() string             { return "SignETHData" }
func (*SignSolData) messageType() string             { return "SignSolData" }
func (*SendETHTx) messageType() string               { return "SendETHTx" }
func (*SendETHTxResponse) messageType() string       { return "SendETHTxResponse" }
func (*SendSolTx) messageType() string               { return "SendSolTx" }
func (*SendSolTxResponse) messageType() string       { return "SendSolTxResponse" }
func (*SignResponse) messageType() string            { return "SignResponse" }
func (*Success) messageType() string                 { return "Success" }
func (*OrderBookSnapshot) messageType() string       { return "OrderBookSnapshot" }
func (*OrderBookDelta) messageType() string          { return "OrderBookDelta" }
func (*CreateOrder) messageType() string             { return "CreateOrder" }
func (*CreateOrderAck) messageType() string          { return "CreateOrderAck" }
func (*CancelOrder) messageType() string             { return "CancelOrder" }
func (*CancelOrderAck) messageType() string          { return "CancelOrderAck" }
func (*AcceptOrder) messageType() string             { return "AcceptOrder" }
func (*AcceptOrderAck) messageType() string          { return "AcceptOrderAck" }
func (*GetOrderBook) messageType() string            { return "GetOrderBook" }
func (*GetOrderBookResponse) messageType() string    { return "GetOrderBookResponse" }
func (*FundingError) messageType() string            { return "FundingError" }
func (*Error) messageType() string                   { return "Error" }
func (*MockMessage) messageType() string             { return "MockMessage" }
func (*SendSignedState) messageType() string         { return "SendSignedState" }

// NewRequest creates a new Request with the given ID and Message.
func NewRequest(ID uint64, msg Message) *Request {
	return &Request{
		ID:      ID,
		Message: &JSONObject{Message: msg},
	}
}

// NewResponse creates a new Response with the given ID and Message.
func NewResponse(ID uint64, msg Message) *Response {
	return &Response{
		ID:      ID,
		Message: &JSONObject{msg},
	}
}

// NewFundingError creates a new FundingError with the given ProposalID, ChannelID and error.
func NewFundingError(propID client.ProposalID, chID channel.ID, err error) *FundingError {
	return &FundingError{
		ProposalID: propID,
		ChannelID:  chID,
		Err:        err.Error(),
	}
}

// NewError creates a new Error with the given error.
func NewError(err error) *Error {
	return &Error{Err: err.Error()}
}

func (f *FundingError) Error() string {
	return fmt.Sprintf("funding of channel %x for proposal %x failed with err: %s", f.ChannelID, f.ProposalID, f.Err)
}

func (e *Error) Error() string {
	return e.Err
}

// MarshalJSON marshals a JSONObject into JSON.
func (o *JSONObject) MarshalJSON() ([]byte, error) {
	msg := struct {
		Type    string  `json:"type"`
		Message Message `json:"message"`
	}{
		Type:    o.Message.messageType(),
		Message: o.Message,
	}

	return json.Marshal(msg)
}

// UnmarshalJSON unmarshals a JSONObject from JSON.
func (o *JSONObject) UnmarshalJSON(data []byte) error {
	var msg struct {
		Type    string          `json:"type"`
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	msgType, ok := messageTypes[msg.Type]
	if !ok {
		return fmt.Errorf("message type '%s' not found", msg.Type)
	}

	obj := reflect.New(msgType).Interface()

	msgData, _ := msg.Message.MarshalJSON()
	if err := json.Unmarshal(msgData, obj); err != nil {
		return err
	}

	o.Message = obj.(Message)
	return nil
}

// MarshalJSON marshals SendSignedState into JSON.
func (s SendSignedState) MarshalJSON() ([]byte, error) {
	signedState := SignedState(s)
	json, err := signedState.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json, nil
}

// UnmarshalJSON unmarshals SendSignedState from JSON.
func (s *SendSignedState) UnmarshalJSON(data []byte) error {
	signedState := SignedState{}
	err := json.Unmarshal(data, &signedState)
	if err != nil {
		return err
	}
	s.Params = signedState.Params
	s.State = signedState.State
	s.Sigs = signedState.Sigs
	return nil
}

// ParseClientAddress parses the client address for Ethereum blockchain.
func (msg EthereumInitialize) ParseClientAddress() (interface{}, error) {
	return msg.ClientAddress, nil
}

// ParseClientAddress parses the client address for Solana blockchain.
func (msg SolanaInitialize) ParseClientAddress() (interface{}, error) {
	kp, err := keypair.Parse(msg.ClientAddress)
	if err != nil {
		return nil, err
	}
	return kp, nil
}

// ClientAddresses contains the client addresses for crossChain blockchain.
type ClientAddresses struct {
	SolClientAddress string
	EthClientAddress common.Address
}

// ParseClientAddress parses the client address for crossChain blockchain.
func (msg CrossContractInitialize) ParseClientAddress() (interface{}, error) {
	return ClientAddresses{msg.SolClientAddress, msg.EthClientAddress}, nil
}

// Initialize is the initial message expected to be sent by the WebSocket
// client and runs the Perun client for the given address.
//
// This is the only message sent by the WebSocket client that is not wrapped
// in a Request or Response.
func Initialize(initMsg interface{}) (BlockchainClient, error) {
	switch msg := initMsg.(type) {
	case *EthereumInitialize:
		return msg, nil
	case *SolanaInitialize:
		return msg, nil
	case *CrossContractInitialize:
		return msg, nil
	default:
		return nil, errors.New("unsupported blockchain type")
	}
}
