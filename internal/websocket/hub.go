package websocket

import (
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/perun-network/perun-dex-websocket/internal/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demo purposes only
	},
}

// Hub manages active WebSocket clients and channel messages
type Hub struct {
	clients    map[string]*types.Client
	channels   map[string]*types.ChannelInfo
	swaps      map[string]*types.SwapProposal
	register   chan *types.Client
	unregister chan *types.Client
	mutex      sync.RWMutex

	exchangeRates *types.ExchangeRates
}

// NewHub creates a new WebSocket Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*types.Client),
		channels:   make(map[string]*types.ChannelInfo),
		swaps:      make(map[string]*types.SwapProposal),
		register:   make(chan *types.Client),
		unregister: make(chan *types.Client),
		exchangeRates: &types.ExchangeRates{
			ETHToSOL:  big.NewFloat(50.0), // Mock rate: 1 ETH = 50 SOL
			SOLToETH:  big.NewFloat(0.02),
			UpdatedAt: time.Now(),
		},
	}
}

// Run runs the Hub main event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client.ID] = client
			h.mutex.Unlock()
			log.Printf("[hub] Client %s registered", client.ID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				for _, ch := range client.Channels {
					if !ch.IsSettled {
						log.Printf("[hub] Client %s disconnected with open channel %s", client.ID, ch.ID)
					}
				}
			}
			h.mutex.Unlock()
			log.Printf("[hub] Client %s unregistered", client.ID)
		}
	}
}

// HandleWebSocket upgrades HTTP → WS and starts client listener
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[hub] WebSocket upgrade error: %v", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
		return
	}

	client := &types.Client{
		ID:         uuid.NewString(),
		Connection: conn,
		Channels:   make(map[string]*types.ChannelInfo),
	}

	h.register <- client
	log.Printf("[hub] New connection: %s", client.ID)

	h.handleClientMessages(client)
}

// handleClientMessages processes all incoming client messages
func (h *Hub) handleClientMessages(client *types.Client) {
	defer func() {
		h.unregister <- client
		client.Connection.(*websocket.Conn).Close()
	}()

	conn := client.Connection.(*websocket.Conn)
	for {
		var msg types.WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[hub] Read error from %s: %v", client.ID, err)
			}
			break
		}
		h.processMessage(client, &msg)
	}
}

// processMessage routes messages by type
func (h *Hub) processMessage(client *types.Client, msg *types.WebSocketMessage) {
	switch msg.Type {
	case types.MsgTypeRegister:
		h.handleRegister(client, msg)
	case types.MsgTypeListClients:
		h.handleListClients(client)
	case types.MsgTypeCreateChannel:
		h.handleCreateChannel(client, msg)
	case types.MsgTypeJoinChannel:
		h.handleJoinChannel(client, msg)
	case types.MsgTypeListChannels:
		h.handleListChannels(client)
	case types.MsgTypeSwapPropose:
		h.handleSwapPropose(client, msg)
	case types.MsgTypeSwapAccept:
		h.handleSwapAccept(client, msg)
	case types.MsgTypeSwapReject:
		h.handleSwapReject(client, msg)
	case types.MsgTypeSettle:
		h.handleSettle(client, msg)
	default:
		h.sendError(client, "unknown message type", 400)
	}
}

// sendMessage sends a structured message to a client
func (h *Hub) sendMessage(client *types.Client, msgType types.MessageType, data interface{}) {
	msg := types.WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
	conn := client.Connection.(*websocket.Conn)
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("[hub] Failed to send to %s: %v", client.ID, err)
	}
}

// sendError sends a structured error
func (h *Hub) sendError(client *types.Client, message string, code int) {
	h.sendMessage(client, types.MsgTypeError, types.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// broadcastToChannel sends a message to all channel participants
func (h *Hub) broadcastToChannel(channelID string, msgType types.MessageType, data interface{}) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	payload, _ := json.Marshal(types.WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	})

	for _, client := range h.clients {
		if _, ok := client.Channels[channelID]; ok {
			conn := client.Connection.(*websocket.Conn)
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				log.Printf("[hub] Broadcast error to %s: %v", client.ID, err)
			}
		}
	}
}
