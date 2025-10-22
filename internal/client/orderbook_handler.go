package client

import (
	"github.com/perun-network/perun-dex-websocket/internal/message"
	"github.com/perun-network/perun-dex-websocket/internal/orderbook"
)

// OrderBookEngine is the global order book instance.
var OrderBookEngine = orderbook.NewEngine()

// HandleOrderBookMessage routes order book messages to the engine.
func (h *requestHandler) HandleOrderBookMessage(msg message.Message) (message.Message, bool) {
	switch m := msg.(type) {
	case *message.CreateOrder:
		book := OrderBookEngine.GetOrCreateBook(m.Order.ChannelID)
		ack := book.CreateOrder(m.Order)
		return &ack, true

	case *message.CancelOrder:
		book, ok := OrderBookEngine.GetBook(m.ChannelID)
		if !ok {
			return &message.CancelOrderAck{
				ID:      m.ID,
				Success: false,
				Reason:  "channel not found",
			}, true
		}
		ack := book.CancelOrder(m.ID)
		return &ack, true

	case *message.AcceptOrder:
		book, ok := OrderBookEngine.GetBook(m.ChannelID)
		if !ok {
			return &message.AcceptOrderAck{
				ID:       m.ID,
				Accepted: false,
				Reason:   "channel not found",
			}, true
		}
		ack := book.AcceptOrder(m.ID)
		return &ack, true

	case *message.GetOrderBook:
		book, ok := OrderBookEngine.GetBook(m.ChannelID)
		if !ok {
			empty := message.OrderBookSnapshot{
				ChannelID: m.ChannelID,
				Sequence:  0,
				TotalOpen: 0,
			}
			return &message.GetOrderBookResponse{Snapshot: &empty}, true
		}
		snap := book.Snapshot()
		return &message.GetOrderBookResponse{Snapshot: &snap}, true
	}

	return nil, false
}
