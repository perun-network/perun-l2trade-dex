package client

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/perun-network/perun-dex-websocket/internal/message"
	ethchannel "github.com/perun-network/perun-eth-backend/channel"
	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire"
)

func (c *Client) handleOpenChannel(msg *message.OpenChannel) (err error) {
	if err := c.checkAssets(msg.State.Assets); err != nil {
		return err
	}

	balances, err := message.MakePerunBals(
		msg.State.Balance, msg.State.PeerBalance, 0, 1,
	)
	if msg.State.Assets == nil || msg.State.Backends == nil {
		return errors.New("assets or backends missing")
	}

	as, err := message.MakePerunAssets(msg.State.Assets, msg.State.Backends)
	if err != nil {
		return err
	}
	backends := make([]wallet.BackendID, len(msg.State.Backends))
	for i, b := range msg.State.Backends {
		backends[i] = wallet.BackendID(b)
	}
	initAlloc := channel.Allocation{
		Assets:   as,
		Backends: backends,
		Balances: balances,
	}

	var peer *Client
	var ok bool
	if msg.PeerAddressEth != (common.Address{}) {
		peer, ok = c.reg.Get(msg.PeerAddressEth.String())
		if !ok {
			return errors.New("peer not found")
		}
	}
	if msg.PeerAddressSol != "" {
		peer, ok = c.reg.Get(msg.PeerAddressSol)
		if !ok {
			return errors.New("peer not found")
		}
	}

	if peer == nil {
		err = fmt.Errorf("peer not found")
		return
	}
	peers := []map[wallet.BackendID]wire.Address{
		c.wireAddrs,
		peer.wireAddrs,
	}
	prop, err := client.NewLedgerChannelProposal(
		msg.ChallengeDuration,
		c.addrs,
		&initAlloc,
		peers,
	)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeouts.HandleTimeout+c.Timeouts.FundTimeout)
	defer cancel()
	ch, err := c.perunClient.ProposeChannel(ctx, prop)

	c.channelCreated(ch, err, msg.ProposalID)
	return err
}

func (c *Client) handleUpdateChannel(msg *message.UpdateChannel) (err error) {
	ch, err := c.perunClient.Channel(msg.ID)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeouts.HandleTimeout)
	defer cancel()

	newBals, err := message.MakePerunBals(
		msg.State.Balance, msg.State.PeerBalance, ch.Idx(), 1-ch.Idx(),
	)
	if err != nil {
		return err
	}

	err = ch.Update(ctx, func(s *channel.State) {
		s.Allocation.Balances = newBals
		s.IsFinal = msg.State.IsFinal
	})
	return
}

func (c *Client) handleCloseChannel(msg *message.CloseChannel) (err error) {
	ch, ok := c.getChannel(msg.ID)
	if !ok {
		return errors.Errorf("channel, %x not found", msg.ID)
	}

	// Unless the forceClose flag is set or the channel state is already final,
	// we first propose a final state before settling.
	if !msg.ForceClose && !ch.State().IsFinal {
		ctxUp, cancel := context.WithTimeout(context.Background(),
			c.Timeouts.HandleTimeout)
		defer cancel()

		err = ch.Update(ctxUp, func(state *channel.State) {
			state.IsFinal = true
		})
		if err != nil {
			return
		}
	}

	ctxSettle, cancel := context.WithTimeout(context.Background(),
		c.Timeouts.SettleTimeout)
	defer cancel()

	c.log("Settling channel", ch.ID())
	err = ch.Settle(ctxSettle, false)
	if err != nil {
		return
	}
	c.log(fmt.Sprintf("Settled channel %x", ch.ID()))

	err = ch.Close()
	if err != nil {
		return
	}
	return
}

func (c *Client) changeReceiverAddr(addr common.Address, ch *client.Channel) error {
	chainIDs := make(map[multi.LedgerIDMapKey]multi.LedgerBackendID)
	for _, a := range ch.State().Assets {
		asset := a.(*ethchannel.Asset)
		chainIDs[asset.LedgerBackendID().LedgerID().MapKey()] = asset.LedgerBackendID()
	}

	for _, chainID := range chainIDs {
		adj, ok := c.adjudicator.LedgerAdjudicator(chainID)
		if !ok {
			return fmt.Errorf("adjudicator for chain %s not found", chainID)
		}
		ethAdj := adj.(*ethchannel.Adjudicator)
		ethAdj.Receiver = addr
	}
	return nil
}

type watcherEventHandler struct {
	*Client
}

// HandleAdjudicatorEvent handles the concluded event.
func (h *watcherEventHandler) HandleAdjudicatorEvent(e channel.AdjudicatorEvent) {
	h.log("HandleAdjudicatorEvent", e)
	if _, ok := e.(*channel.ConcludedEvent); ok {
		h.log("Received concluded event")
		ch, ok := h.getChannel(e.ID())
		if !ok { // In this case we already settled and removed the channel.
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(),
			h.Timeouts.SettleTimeout)
		defer cancel()
		err := ch.Settle(ctx, true)
		if err != nil {
			h.log("Error while settling", err)
			return
		}

		h.channelClosed(ch.ID())
	}
}

// channelCreated adds the created channel to the client's channel map, checks
// the error received when opening a channel, and if no error occurred, starts
// the watcher and sends a ChannelCreated message to the client.
func (c *Client) channelCreated(ch *client.Channel, chErr error, propID client.ProposalID) {
	if ch != nil {
		c.log(fmt.Sprintf("Channel created %x", ch.ID()))
		c.addChannel(ch)
	} else {
		return
	}

	if chErr != nil {
		if _, ok := chErr.(*client.ChannelFundingError); ok {
			err := c.conn.Write(message.NewFundingError(propID, ch.ID(), chErr))
			if err != nil {
				c.log("sending funding error message", err)
			}
		}
		return
	}

	go func() {
		err := ch.Watch(&watcherEventHandler{c})
		c.log(fmt.Sprintf("channel %v: watcher returned: %v", ch.ID(), err))
	}()

	peerClient, ok := c.reg.Get(ch.Params().Parts[1-ch.Idx()][message.EthereumIndex].String())
	if !ok {
		c.log("Error getting peer client")
		return
	}

	timeout := time.After(1 * time.Minute)
	tick := time.Tick(500 * time.Millisecond)

	for {
		select {
		case <-timeout:
			c.log(fmt.Sprintf("timeout waiting for peer to create channel %x", ch.ID()))
			err := c.conn.Write(message.NewFundingError(propID, ch.ID(), fmt.Errorf("peer did not create channel in time")))
			if err != nil {
				c.log("sending timeout funding error message", err)
			}
			return
		case <-tick:
			_, ok = peerClient.getChannel(ch.ID())
			if ok {
				msg := &message.ChannelCreated{ID: ch.ID(), ProposalID: propID, Idx: ch.Idx()}
				err := c.conn.Write(msg)
				if err != nil {
					c.log("sending channel created message", err)
				}
				return
			}
		}
	}
}

// channelClosed removes the channel out of the client's channel map and sends a
// ChannelClosed message to the client.
func (c *Client) channelClosed(chID channel.ID) {
	c.log(fmt.Sprintf("Closed channel %x", chID))
	c.removeChannel(chID)

	err := c.conn.Write(&message.ChannelClosed{ID: chID})
	if err != nil {
		c.log("sending channel closed message")
	}
}

func (c *Client) channelProposal(lcp *client.LedgerChannelProposalMsg) (*message.ProposalResponse, error) {
	if lcp.NumPeers() != 2 {
		panic(fmt.Sprintf("expected number of peers 2, got %v", lcp.NumPeers()))
	}

	var proposerBalIndex, proposeeBalIndex channel.Index = 0, 1
	return c.conn.ChannelProposal(lcp, proposeeBalIndex, proposerBalIndex)
}
