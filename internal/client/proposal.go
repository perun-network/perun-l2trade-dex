package client

import (
	"context"
	"fmt"

	"github.com/perun-network/perun-dex-websocket/internal/message"
	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	"perun.network/go-perun/log"
)

func (c *Client) handleProposals() (err error) {
	handler := &ProposalHandler{c}
	c.perunClient.Handle(handler, handler)
	return fmt.Errorf("unknown error")
}

// ProposalHandler handles channel proposals.
type ProposalHandler struct {
	*Client
}

// HandleProposal handles a channel proposal.
func (h *ProposalHandler) HandleProposal(p client.ChannelProposal, r *client.ProposalResponder) {
	h.handleChannelProposal(p, r)
}

func (c *Client) handleChannelProposal(p client.ChannelProposal, r *client.ProposalResponder) {
	err := func() (err error) {
		lcp, ok := p.(*client.LedgerChannelProposalMsg)
		if !ok {
			err = fmt.Errorf("expected ledger channel proposal, got %T", p)
			return
		}

		if len(lcp.Peers) != 2 {
			err = fmt.Errorf("only two participant channels supported, got %d", len(lcp.Peers))
			return
		}

		assets := message.MakeAssetsGPAsAssets(lcp.InitBals.Assets)
		fmt.Println("Proposal Assets: ", assets)
		if err = c.checkAssets(assets); err != nil {
			c.log(err)
			return
		}
		log.Println("Proposal: ", lcp)
		resp, err := c.channelProposal(lcp)
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.Timeouts.FundTimeout)
		defer cancel()
		if resp.Accepted {
			acceptance := lcp.Accept(c.addrs, client.WithRandomNonce())
			var ch *client.Channel
			ch, err = r.Accept(ctx, acceptance)
			if err != nil {
				log.Println(err)
				return
			}
			c.channelCreated(ch, err, lcp.ProposalID)
		} else {
			err = r.Reject(ctx, resp.RejectReason)
		}
		return
	}()
	if err != nil {
		log.Error(errors.Wrap(err, "handling channel proposal"))
	}
}

// HandleUpdate handles a channel update.
func (h *ProposalHandler) HandleUpdate(s *channel.State, u client.ChannelUpdate, r *client.UpdateResponder) {
	h.handleUpdateProposal(s, u, r)
}

func (c *Client) handleUpdateProposal(s *channel.State, u client.ChannelUpdate, r *client.UpdateResponder) {
	err := func() (err error) {
		accepted, reason, err := c.updateProposal(s, u)
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.Timeouts.DefaultTimeout)
		defer cancel()
		if accepted {
			err = r.Accept(ctx)
			if err != nil {
				return
			}
		} else {
			err = r.Reject(ctx, reason)
			if err != nil {
				return
			}
		}
		return
	}()
	if err != nil {
		c.CloseWithError(err)
	}
}

func (c *Client) updateProposal(s *channel.State, u client.ChannelUpdate) (accepted bool, reason string, err error) {
	if u.State.NumParts() != 2 {
		panic(fmt.Sprintf("expected number of participants 2, got %v", u.State.NumParts()))
	}

	ch, err := c.perunClient.Channel(u.State.ID)
	if err != nil {
		return
	}

	myIdx := ch.Idx()
	peerIdx := 1 - myIdx
	return c.conn.UpdateProposal(s, u, myIdx, peerIdx)
}
