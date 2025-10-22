package message

import "perun.network/go-perun/channel"

// OrderSide represents bid or ask for an order.
type OrderSide string

const (
	SideBid OrderSide = "bid"
	SideAsk OrderSide = "ask"
)

// OrderStatus captures order lifecycle on the off-chain order book.
type OrderStatus string

const (
	OrderOpen     OrderStatus = "open"
	OrderAccepted OrderStatus = "accepted"
	OrderCanceled OrderStatus = "canceled"
	OrderRejected OrderStatus = "rejected"
	OrderFilled   OrderStatus = "filled"
)

// OrderID is the unique identifier of an off-chain order.
type OrderID string

// Order describes a limit order proposal for a Perun ledger channel.
// Amounts are strings in base units of the base asset; price is quoted as
// a rational on-chain price in quote/base, represented as a decimal string.
type Order struct {
	ID        OrderID       `json:"id"`
	ChannelID channel.ID    `json:"channelID"`
	MakerIdx  channel.Index `json:"makerIdx"`  // who created it, 0/1
	Side      OrderSide     `json:"side"`      // "bid" or "ask"
	Base      Asset         `json:"base"`      // asset being bought/sold
	Quote     Asset         `json:"quote"`     // pricing asset
	Price     string        `json:"price"`     // decimal string
	Amount    string        `json:"amount"`    // base units
	Status    OrderStatus   `json:"status"`    // lifecycle status
	CreatedAt int64         `json:"createdAt"` // unix seconds
	ExpiresAt *int64        `json:"expiresAt,omitempty"`
	ClientTag string        `json:"clientTag,omitempty"` // optional client tag
}

// OrderBookSnapshot provides a full view of current active orders for a channel.
type OrderBookSnapshot struct {
	ChannelID channel.ID `json:"channelID"`
	Sequence  uint64     `json:"sequence"`  // snapshot sequence
	TotalOpen uint64     `json:"totalOpen"` // active orders count
	Bids      []Order    `json:"bids"`      // open bid orders
	Asks      []Order    `json:"asks"`      // open ask orders
}

// OrderBookDelta provides incremental changes since the previous sequence.
type OrderBookDelta struct {
	ChannelID channel.ID `json:"channelID"`
	Sequence  uint64     `json:"sequence"` // strictly increasing
	// Added/Updated are full rows; Removed are order IDs.
	Added     []Order   `json:"added"`
	Updated   []Order   `json:"updated"`
	Removed   []OrderID `json:"removed"`
	TotalOpen uint64    `json:"totalOpen"` // after applying this delta
}
