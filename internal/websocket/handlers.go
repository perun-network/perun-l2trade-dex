package websocket

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/perun-network/perun-dex-websocket/internal/types"
)

// handleRegister processes client registration
func (h *Hub) handleRegister(client *types.Client, msg *types.WebSocketMessage) {
	var registerMsg types.RegisterMessage
	data, err := json.Marshal(msg.Data)
	if err != nil {
		h.sendError(client, "Invalid registration data", 400)
		return
	}

	err = json.Unmarshal(data, &registerMsg)
	if err != nil {
		h.sendError(client, "Invalid registration format", 400)
		return
	}

	// Update client information
	client.UserID = registerMsg.UserID
	client.Username = registerMsg.Username
	client.Address = registerMsg.Address

	// Send registration response
	response := types.RegisterResponse{
		UserID:    client.UserID,
		Username:  client.Username,
		Address:   client.Address,
		Connected: true,
	}

	h.sendMessage(client, types.MsgTypeRegisterResp, response)
}

// handleListClients processes requests to list all connected clients
func (h *Hub) handleListClients(client *types.Client, msg *types.WebSocketMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var clients []types.ClientInfo
	for _, c := range h.clients {
		clients = append(clients, types.ClientInfo{
			UserID:   c.UserID,
			Username: c.Username,
			Address:  c.Address,
			Online:   true,
		})
	}

	response := types.ListClientsResponse{
		Clients: clients,
	}

	h.sendMessage(client, types.MsgTypeClientsResp, response)
}

// handleListChannels processes requests to list user's channels
func (h *Hub) handleListChannels(client *types.Client, msg *types.WebSocketMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var channels []types.ChannelSummary
	for _, channel := range client.Channels {
		// Convert balances to strings for response
		balanceStrs := make(map[string]map[types.AssetType]string)
		for userID, userBalances := range channel.Balances {
			balanceStrs[userID] = make(map[types.AssetType]string)
			for asset, balance := range userBalances {
				balanceStrs[userID][asset] = balance.String()
			}
		}

		channels = append(channels, types.ChannelSummary{
			ChannelID:    channel.ChannelID,
			Participants: channel.Participants,
			Balances:     balanceStrs,
			IsSettled:    channel.IsSettled,
			CreatedAt:    channel.CreatedAt,
		})
	}

	response := types.ListChannelsResponse{
		Channels: channels,
	}

	h.sendMessage(client, types.MsgTypeChannelsResp, response)
}

// handleCreateChannel processes channel creation requests
func (h *Hub) handleCreateChannel(client *types.Client, msg *types.WebSocketMessage) {
	var createMsg types.CreateChannelMessage
	data, err := json.Marshal(msg.Data)
	if err != nil {
		h.sendError(client, "Invalid channel creation data", 400)
		return
	}

	err = json.Unmarshal(data, &createMsg)
	if err != nil {
		h.sendError(client, "Invalid channel creation format", 400)
		return
	}

	// Validate participant count (at least 2 including creator)
	if len(createMsg.ParticipantUserIDs) < 1 {
		h.sendError(client, "At least one other participant required", 400)
		return
	}

	// Find all participant clients
	h.mutex.RLock()
	var participantClients []*types.Client
	participantClients = append(participantClients, client) // Add creator

	for _, userID := range createMsg.ParticipantUserIDs {
		found := false
		for _, c := range h.clients {
			if c.UserID == userID {
				participantClients = append(participantClients, c)
				found = true
				break
			}
		}
		if !found {
			h.mutex.RUnlock()
			h.sendError(client, fmt.Sprintf("Participant %s not found or not connected", userID), 404)
			return
		}
	}
	h.mutex.RUnlock()

	// Create new channel
	channelID := uuid.New().String()

	// Initialize balances for all participants
	participantBalances := make(map[string]map[types.AssetType]*big.Int)
	for _, participant := range participantClients {
		participantBalances[participant.UserID] = make(map[types.AssetType]*big.Int)
		participantBalances[participant.UserID][types.AssetETH] = big.NewInt(0)
		participantBalances[participant.UserID][types.AssetSOL] = big.NewInt(0)
	}

	// Set initial balances from request
	for userID, userBalances := range createMsg.InitialBalances {
		if participantBalances[userID] == nil {
			h.sendError(client, fmt.Sprintf("User %s is not a participant", userID), 400)
			return
		}
		for asset, balanceStr := range userBalances {
			balance, ok := new(big.Int).SetString(balanceStr, 10)
			if !ok {
				h.sendError(client, fmt.Sprintf("Invalid balance for %s:%s", userID, asset), 400)
				return
			}
			participantBalances[userID][asset] = balance
		}
	}

	// Get participant user IDs
	participantUserIDs := make([]string, len(participantClients))
	for i, participant := range participantClients {
		participantUserIDs[i] = participant.UserID
	}

	channelInfo := &types.ChannelInfo{
		ID:           channelID,
		ChannelID:    channelID,
		Participants: participantUserIDs,
		Balances:     participantBalances,
		IsSettled:    false,
		CreatedAt:    time.Now(),
	}

	// Store channel and add to all participants
	h.mutex.Lock()
	h.channels[channelID] = channelInfo
	for _, participant := range participantClients {
		participant.Channels[channelID] = channelInfo
	}
	h.mutex.Unlock()

	// Convert balances to strings for response
	balanceStrs := make(map[string]map[types.AssetType]string)
	for userID, userBalances := range participantBalances {
		balanceStrs[userID] = make(map[types.AssetType]string)
		for asset, balance := range userBalances {
			balanceStrs[userID][asset] = balance.String()
		}
	}

	// Send response to all participants
	response := types.ChannelCreatedMessage{
		ChannelID:    channelID,
		Participants: participantUserIDs,
		Balances:     balanceStrs,
		State:        "created",
	}

	for _, participant := range participantClients {
		h.sendMessage(participant, types.MsgTypeChannelCreated, response)
	}
}

// handleJoinChannel processes channel join requests
func (h *Hub) handleJoinChannel(client *types.Client, msg *types.WebSocketMessage) {
	var joinMsg types.JoinChannelMessage
	data, err := json.Marshal(msg.Data)
	if err != nil {
		h.sendError(client, "Invalid join channel data", 400)
		return
	}

	err = json.Unmarshal(data, &joinMsg)
	if err != nil {
		h.sendError(client, "Invalid join channel format", 400)
		return
	}

	h.mutex.Lock()
	channel, exists := h.channels[joinMsg.ChannelID]
	if !exists {
		h.mutex.Unlock()
		h.sendError(client, "Channel not found", 404)
		return
	}

	client.Channels[joinMsg.ChannelID] = channel
	h.mutex.Unlock()

	// Convert balances to strings for response
	balanceStrs := make(map[string]map[types.AssetType]string)
	for userID, userBalances := range channel.Balances {
		balanceStrs[userID] = make(map[types.AssetType]string)
		for asset, balance := range userBalances {
			balanceStrs[userID][asset] = balance.String()
		}
	}

	response := types.ChannelCreatedMessage{
		ChannelID:    joinMsg.ChannelID,
		Participants: channel.Participants,
		Balances:     balanceStrs,
		State:        "joined",
	}

	h.sendMessage(client, types.MsgTypeChannelJoined, response)
}

// handleSwapPropose processes swap proposals
func (h *Hub) handleSwapPropose(client *types.Client, msg *types.WebSocketMessage) {
	var swapMsg types.SwapProposeMessage
	data, err := json.Marshal(msg.Data)
	if err != nil {
		h.sendError(client, "Invalid swap proposal data", 400)
		return
	}

	err = json.Unmarshal(data, &swapMsg)
	if err != nil {
		h.sendError(client, "Invalid swap proposal format", 400)
		return
	}

	// Validate channel exists and client is participant
	h.mutex.RLock()
	channel, exists := h.channels[swapMsg.ChannelID]
	if !exists {
		h.mutex.RUnlock()
		h.sendError(client, "Channel not found", 404)
		return
	}

	// Check if client is participant
	isParticipant := false
	for _, userID := range channel.Participants {
		if userID == client.UserID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		h.mutex.RUnlock()
		h.sendError(client, "Not a participant in this channel", 403)
		return
	}
	h.mutex.RUnlock()

	// Parse amounts
	fromAmount, ok := new(big.Int).SetString(swapMsg.FromAmount, 10)
	if !ok {
		h.sendError(client, "Invalid from amount", 400)
		return
	}

	toAmount, ok := new(big.Int).SetString(swapMsg.ToAmount, 10)
	if !ok {
		h.sendError(client, "Invalid to amount", 400)
		return
	}

	exchangeRate, ok := new(big.Float).SetString(swapMsg.ExchangeRate)
	if !ok {
		h.sendError(client, "Invalid exchange rate", 400)
		return
	}

	// Create swap proposal
	swapID := uuid.New().String()
	proposal := &types.SwapProposal{
		ID:           swapID,
		ChannelID:    swapMsg.ChannelID,
		Proposer:     client.UserID,
		FromAsset:    swapMsg.FromAsset,
		ToAsset:      swapMsg.ToAsset,
		FromAmount:   fromAmount,
		ToAmount:     toAmount,
		ExchangeRate: exchangeRate,
		Timestamp:    time.Now(),
		Status:       types.SwapStatusPending,
	}

	// Store proposal
	h.mutex.Lock()
	h.swaps[swapID] = proposal
	h.mutex.Unlock()

	// Broadcast to channel participants
	proposalData := map[string]interface{}{
		"swap_id":       swapID,
		"channel_id":    swapMsg.ChannelID,
		"proposer":      client.UserID,
		"from_asset":    swapMsg.FromAsset,
		"to_asset":      swapMsg.ToAsset,
		"from_amount":   swapMsg.FromAmount,
		"to_amount":     swapMsg.ToAmount,
		"exchange_rate": swapMsg.ExchangeRate,
	}

	h.broadcastToChannel(swapMsg.ChannelID, types.MsgTypeSwapPropose, proposalData)
}

// handleSwapAccept processes swap acceptance
func (h *Hub) handleSwapAccept(client *types.Client, msg *types.WebSocketMessage) {
	var acceptMsg types.SwapAcceptMessage
	data, err := json.Marshal(msg.Data)
	if err != nil {
		h.sendError(client, "Invalid swap accept data", 400)
		return
	}

	err = json.Unmarshal(data, &acceptMsg)
	if err != nil {
		h.sendError(client, "Invalid swap accept format", 400)
		return
	}

	h.mutex.Lock()
	proposal, exists := h.swaps[acceptMsg.SwapID]
	if !exists {
		h.mutex.Unlock()
		h.sendError(client, "Swap proposal not found", 404)
		return
	}

	if proposal.Status != types.SwapStatusPending {
		h.mutex.Unlock()
		h.sendError(client, "Swap proposal is no longer pending", 400)
		return
	}

	// Update proposal status
	proposal.Status = types.SwapStatusAccepted

	// Get channel and update balances
	channel := h.channels[proposal.ChannelID]

	// Find proposer and acceptor
	proposerUserID := proposal.Proposer
	acceptorUserID := client.UserID

	// Initialize balances if needed
	if channel.Balances[proposerUserID] == nil {
		channel.Balances[proposerUserID] = make(map[types.AssetType]*big.Int)
	}
	if channel.Balances[acceptorUserID] == nil {
		channel.Balances[acceptorUserID] = make(map[types.AssetType]*big.Int)
	}

	// Initialize asset balances if needed
	if channel.Balances[proposerUserID][proposal.FromAsset] == nil {
		channel.Balances[proposerUserID][proposal.FromAsset] = big.NewInt(0)
	}
	if channel.Balances[proposerUserID][proposal.ToAsset] == nil {
		channel.Balances[proposerUserID][proposal.ToAsset] = big.NewInt(0)
	}
	if channel.Balances[acceptorUserID][proposal.FromAsset] == nil {
		channel.Balances[acceptorUserID][proposal.FromAsset] = big.NewInt(0)
	}
	if channel.Balances[acceptorUserID][proposal.ToAsset] == nil {
		channel.Balances[acceptorUserID][proposal.ToAsset] = big.NewInt(0)
	}

	// Update balances: proposer gives FromAmount of FromAsset, gets ToAmount of ToAsset
	// Acceptor gives ToAmount of ToAsset, gets FromAmount of FromAsset
	channel.Balances[proposerUserID][proposal.FromAsset].Sub(
		channel.Balances[proposerUserID][proposal.FromAsset],
		proposal.FromAmount,
	)
	channel.Balances[proposerUserID][proposal.ToAsset].Add(
		channel.Balances[proposerUserID][proposal.ToAsset],
		proposal.ToAmount,
	)
	channel.Balances[acceptorUserID][proposal.FromAsset].Add(
		channel.Balances[acceptorUserID][proposal.FromAsset],
		proposal.FromAmount,
	)
	channel.Balances[acceptorUserID][proposal.ToAsset].Sub(
		channel.Balances[acceptorUserID][proposal.ToAsset],
		proposal.ToAmount,
	)

	proposal.Status = types.SwapStatusCompleted
	h.mutex.Unlock()

	// Convert balances to strings for response
	newBalances := make(map[string]map[types.AssetType]string)
	for userID, userBalances := range channel.Balances {
		newBalances[userID] = make(map[types.AssetType]string)
		for asset, balance := range userBalances {
			newBalances[userID][asset] = balance.String()
		}
	}

	// Broadcast completion
	completeData := types.SwapCompleteMessage{
		ChannelID:     proposal.ChannelID,
		SwapID:        acceptMsg.SwapID,
		NewBalances:   newBalances,
		TransactionID: uuid.New().String(), // Mock transaction ID
	}

	h.broadcastToChannel(proposal.ChannelID, types.MsgTypeSwapComplete, completeData)
}

// handleSwapReject processes swap rejection
func (h *Hub) handleSwapReject(client *types.Client, msg *types.WebSocketMessage) {
	var rejectMsg types.SwapRejectMessage
	data, err := json.Marshal(msg.Data)
	if err != nil {
		h.sendError(client, "Invalid swap reject data", 400)
		return
	}

	err = json.Unmarshal(data, &rejectMsg)
	if err != nil {
		h.sendError(client, "Invalid swap reject format", 400)
		return
	}

	h.mutex.Lock()
	proposal, exists := h.swaps[rejectMsg.SwapID]
	if !exists {
		h.mutex.Unlock()
		h.sendError(client, "Swap proposal not found", 404)
		return
	}

	proposal.Status = types.SwapStatusRejected
	h.mutex.Unlock()

	// Broadcast rejection
	h.broadcastToChannel(proposal.ChannelID, types.MsgTypeSwapReject, rejectMsg)
}

// handleSettle processes channel settlement
func (h *Hub) handleSettle(client *types.Client, msg *types.WebSocketMessage) {
	var settleMsg types.SettleMessage
	data, err := json.Marshal(msg.Data)
	if err != nil {
		h.sendError(client, "Invalid settle data", 400)
		return
	}

	err = json.Unmarshal(data, &settleMsg)
	if err != nil {
		h.sendError(client, "Invalid settle format", 400)
		return
	}

	h.mutex.Lock()
	channel, exists := h.channels[settleMsg.ChannelID]
	if !exists {
		h.mutex.Unlock()
		h.sendError(client, "Channel not found", 404)
		return
	}

	if channel.IsSettled {
		h.mutex.Unlock()
		h.sendError(client, "Channel already settled", 400)
		return
	}

	// Mark as settled
	channel.IsSettled = true
	h.mutex.Unlock()

	// Convert final balances to strings
	finalBalances := make(map[string]map[types.AssetType]string)
	for userID, userBalances := range channel.Balances {
		finalBalances[userID] = make(map[types.AssetType]string)
		for asset, balance := range userBalances {
			finalBalances[userID][asset] = balance.String()
		}
	}

	// Broadcast settlement
	settledData := types.SettledMessage{
		ChannelID:       settleMsg.ChannelID,
		FinalBalances:   finalBalances,
		TransactionHash: fmt.Sprintf("0x%s", uuid.New().String()), // Mock transaction hash
	}

	h.broadcastToChannel(settleMsg.ChannelID, types.MsgTypeSettled, settledData)
}
