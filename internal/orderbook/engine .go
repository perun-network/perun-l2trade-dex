package orderbook

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/perun-network/perun-dex-websocket/internal/message"
	"perun.network/go-perun/channel"
)

// Engine manages all per-channel order books.
type Engine struct {
	mu    sync.RWMutex
	books map[channel.ID]*Book
}

// NewEngine creates a new order book engine.
func NewEngine() *Engine {
	return &Engine{
		books: make(map[channel.ID]*Book),
	}
}

// GetOrCreateBook returns or creates a book for a channel.
func (e *Engine) GetOrCreateBook(chID channel.ID) *Book {
	e.mu.Lock()
	defer e.mu.Unlock()

	b, ok := e.books[chID]
	if !ok {
		b = newBook(chID)
		e.books[chID] = b
	}
	return b
}

// GetBook returns an existing book if it exists.
func (e *Engine) GetBook(chID channel.ID) (*Book, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	b, ok := e.books[chID]
	return b, ok
}

// Book represents a per-channel order book.
type Book struct {
	chID      channel.ID
	mu        sync.Mutex
	sequence  uint64
	totalOpen uint64
	bids      map[message.OrderID]message.Order
	asks      map[message.OrderID]message.Order

	// Subscribers for broadcasting
	subscribers map[chan []byte]bool
	subMu       sync.RWMutex
}

func newBook(chID channel.ID) *Book {
	return &Book{
		chID:        chID,
		bids:        make(map[message.OrderID]message.Order),
		asks:        make(map[message.OrderID]message.Order),
		subscribers: make(map[chan []byte]bool),
	}
}

// Subscribe adds a channel to receive delta broadcasts.
func (b *Book) Subscribe(ch chan []byte) {
	b.subMu.Lock()
	defer b.subMu.Unlock()
	b.subscribers[ch] = true
}

// Unsubscribe removes a subscriber channel.
func (b *Book) Unsubscribe(ch chan []byte) {
	b.subMu.Lock()
	defer b.subMu.Unlock()
	delete(b.subscribers, ch)
	close(ch)
}

// broadcast sends a delta to all subscribers.
func (b *Book) broadcast(delta message.OrderBookDelta) {
	frame := message.JSONObject{Message: &delta}
	data, _ := json.Marshal(frame)

	b.subMu.RLock()
	defer b.subMu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- data:
		default:
			// Non-blocking send; drop slow consumers
		}
	}
}

// Snapshot returns current state.
func (b *Book) Snapshot() message.OrderBookSnapshot {
	b.mu.Lock()
	defer b.mu.Unlock()

	bids := make([]message.Order, 0, len(b.bids))
	for _, o := range b.bids {
		bids = append(bids, o)
	}

	asks := make([]message.Order, 0, len(b.asks))
	for _, o := range b.asks {
		asks = append(asks, o)
	}

	return message.OrderBookSnapshot{
		ChannelID: b.chID,
		Sequence:  b.sequence,
		TotalOpen: b.totalOpen,
		Bids:      bids,
		Asks:      asks,
	}
}

// CreateOrder adds an order and broadcasts delta.
func (b *Book) CreateOrder(o message.Order) message.CreateOrderAck {
	b.mu.Lock()
	defer b.mu.Unlock()

	o.Status = message.OrderOpen
	if o.CreatedAt == 0 {
		o.CreatedAt = time.Now().Unix()
	}

	switch o.Side {
	case message.SideBid:
		b.bids[o.ID] = o
	case message.SideAsk:
		b.asks[o.ID] = o
	default:
		return message.CreateOrderAck{
			ID:       o.ID,
			Accepted: false,
			Reason:   "invalid side",
		}
	}

	b.totalOpen++
	b.sequence++

	delta := message.OrderBookDelta{
		ChannelID: b.chID,
		Sequence:  b.sequence,
		Added:     []message.Order{o},
		TotalOpen: b.totalOpen,
	}

	go b.broadcast(delta)

	return message.CreateOrderAck{
		ID:        o.ID,
		Accepted:  true,
		TotalOpen: b.totalOpen,
	}
}

// CancelOrder removes an order and broadcasts delta.
func (b *Book) CancelOrder(id message.OrderID) message.CancelOrderAck {
	b.mu.Lock()
	defer b.mu.Unlock()

	var found bool
	if _, ok := b.bids[id]; ok {
		delete(b.bids, id)
		found = true
	} else if _, ok := b.asks[id]; ok {
		delete(b.asks, id)
		found = true
	}

	if !found {
		return message.CancelOrderAck{
			ID:      id,
			Success: false,
			Reason:  "order not found",
		}
	}

	if b.totalOpen > 0 {
		b.totalOpen--
	}
	b.sequence++

	delta := message.OrderBookDelta{
		ChannelID: b.chID,
		Sequence:  b.sequence,
		Removed:   []message.OrderID{id},
		TotalOpen: b.totalOpen,
	}

	go b.broadcast(delta)

	return message.CancelOrderAck{
		ID:        id,
		Success:   true,
		TotalOpen: b.totalOpen,
	}
}

// AcceptOrder validates order exists for trading.
func (b *Book) AcceptOrder(id message.OrderID) message.AcceptOrderAck {
	b.mu.Lock()
	defer b.mu.Unlock()

	_, inBids := b.bids[id]
	_, inAsks := b.asks[id]

	if !inBids && !inAsks {
		return message.AcceptOrderAck{
			ID:       id,
			Accepted: false,
			Reason:   "order not found",
		}
	}

	return message.AcceptOrderAck{
		ID:       id,
		Accepted: true,
	}
}
