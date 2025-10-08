package types

import (
	"math/big"
	"time"
)

// MessageType defines the type of WebSocket messages
type MessageType string

const (
	// Client registration and initialization
	MsgTypeRegister     MessageType = "register"
	MsgTypeRegisterResp MessageType = "register_response"
	MsgTypeListClients  MessageType = "list_clients"
	MsgTypeClientsResp  MessageType = "clients_response"

	// Channel operations
	MsgTypeCreateChannel  MessageType = "create_channel"
	MsgTypeChannelCreated MessageType = "channel_created"
	MsgTypeJoinChannel    MessageType = "join_channel"
	MsgTypeChannelJoined  MessageType = "channel_joined"
	MsgTypeListChannels   MessageType = "list_channels"
	MsgTypeChannelsResp   MessageType = "channels_response"

	// Swap operations
	MsgTypeSwapPropose  MessageType = "swap_propose"
	MsgTypeSwapAccept   MessageType = "swap_accept"
	MsgTypeSwapReject   MessageType = "swap_reject"
	MsgTypeSwapComplete MessageType = "swap_complete"

	// Settlement
	MsgTypeSettle  MessageType = "settle"
	MsgTypeSettled MessageType = "settled"

	// General responses
	MsgTypeError   MessageType = "error"
	MsgTypeSuccess MessageType = "success"
)

// AssetType represents supported assets
type AssetType string

const (
	AssetETH AssetType = "ETH"
	AssetSOL AssetType = "SOL"
)

// WebSocketMessage represents the base message structure
type WebSocketMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// RegisterMessage for client registration
type RegisterMessage struct {
	UserID   string `json:"user_id"`
	Address  string `json:"address"`
	Username string `json:"username"`
}

// RegisterResponse after successful registration
type RegisterResponse struct {
	UserID    string `json:"user_id"`
	Address   string `json:"address"`
	Username  string `json:"username"`
	Connected bool   `json:"connected"`
}

// CreateChannelMessage for channel creation
type CreateChannelMessage struct {
	ParticipantUserIDs []string                        `json:"participant_user_ids"`
	InitialBalances    map[string]map[AssetType]string `json:"initial_balances"` // userID -> asset -> amount
	ChallengeDuration  uint64                          `json:"challenge_duration"`
}

// ChannelCreatedMessage response
type ChannelCreatedMessage struct {
	ChannelID    string                          `json:"channel_id"`
	Participants []string                        `json:"participants"`
	Balances     map[string]map[AssetType]string `json:"balances"` // userID -> asset -> amount
	State        string                          `json:"state"`
}

// JoinChannelMessage for joining an existing channel
type JoinChannelMessage struct {
	ChannelID string `json:"channel_id"`
}

// ClientInfo represents basic info about a connected client
type ClientInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Address  string `json:"address"`
	Online   bool   `json:"online"`
}

// ListClientsResponse contains list of connected clients
type ListClientsResponse struct {
	Clients []ClientInfo `json:"clients"`
}

// ChannelSummary represents basic channel information
type ChannelSummary struct {
	ChannelID    string                          `json:"channel_id"`
	Participants []string                        `json:"participants"`
	Balances     map[string]map[AssetType]string `json:"balances"`
	IsSettled    bool                            `json:"is_settled"`
	CreatedAt    time.Time                       `json:"created_at"`
}

// ListChannelsResponse contains list of channels for a user
type ListChannelsResponse struct {
	Channels []ChannelSummary `json:"channels"`
}

// SwapProposeMessage for proposing a swap
type SwapProposeMessage struct {
	ChannelID    string    `json:"channel_id"`
	FromAsset    AssetType `json:"from_asset"`
	ToAsset      AssetType `json:"to_asset"`
	FromAmount   string    `json:"from_amount"`
	ToAmount     string    `json:"to_amount"`
	ExchangeRate string    `json:"exchange_rate"`
}

// SwapAcceptMessage for accepting a swap
type SwapAcceptMessage struct {
	ChannelID string `json:"channel_id"`
	SwapID    string `json:"swap_id"`
}

// SwapRejectMessage for rejecting a swap
type SwapRejectMessage struct {
	ChannelID string `json:"channel_id"`
	SwapID    string `json:"swap_id"`
	Reason    string `json:"reason"`
}

// SwapCompleteMessage when swap is completed
type SwapCompleteMessage struct {
	ChannelID     string                          `json:"channel_id"`
	SwapID        string                          `json:"swap_id"`
	NewBalances   map[string]map[AssetType]string `json:"new_balances"` // userID -> asset -> balance
	TransactionID string                          `json:"transaction_id"`
}

// SettleMessage for channel settlement
type SettleMessage struct {
	ChannelID string `json:"channel_id"`
}

// SettledMessage when channel is settled
type SettledMessage struct {
	ChannelID       string                          `json:"channel_id"`
	FinalBalances   map[string]map[AssetType]string `json:"final_balances"` // userID -> asset -> balance
	TransactionHash string                          `json:"transaction_hash"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID          string
	UserID      string
	Username    string
	Address     string      // wallet address as string
	Connection  interface{} // WebSocket connection
	PerunClient interface{} // Perun client instance
	Channels    map[string]*ChannelInfo
}

// ChannelInfo holds information about a payment channel
type ChannelInfo struct {
	ID           string
	ChannelID    string                            // channel ID as string
	Participants []string                          // participant user IDs
	Balances     map[string]map[AssetType]*big.Int // userID -> asset -> balance
	IsSettled    bool
	CreatedAt    time.Time
}

// SwapProposal represents a pending swap proposal
type SwapProposal struct {
	ID           string
	ChannelID    string
	Proposer     string
	FromAsset    AssetType
	ToAsset      AssetType
	FromAmount   *big.Int
	ToAmount     *big.Int
	ExchangeRate *big.Float
	Timestamp    time.Time
	Status       SwapStatus
}

// SwapStatus represents the status of a swap
type SwapStatus string

const (
	SwapStatusPending   SwapStatus = "pending"
	SwapStatusAccepted  SwapStatus = "accepted"
	SwapStatusRejected  SwapStatus = "rejected"
	SwapStatusCompleted SwapStatus = "completed"
)

// ErrorResponse for error messages
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ExchangeRates holds current exchange rates
type ExchangeRates struct {
	ETHToSOL  *big.Float `json:"eth_to_sol"`
	SOLToETH  *big.Float `json:"sol_to_eth"`
	UpdatedAt time.Time  `json:"updated_at"`
}
