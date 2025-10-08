package message

import (
	"fmt"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
)

// UpdateProposal is a request to update a channel proposal.
func (c *Connection) UpdateProposal(
	s *channel.State,
	u client.ChannelUpdate,
	myIdx, peerIdx channel.Index,
) (ok bool, reason string, err error) {
	myBals, peerBals := MakeBals(u.State.Balances, myIdx, peerIdx)
	state := NewChannelState(
		s.Assets,
		s.Backends,
		myBals,
		peerBals,
		u.State.IsFinal,
	)

	req := &UpdateChannel{
		ID:    u.State.ID,
		State: state,
	}
	resp, err := c.Request(req)
	if err != nil {
		return
	}

	_resp, ok := resp.(*ProposalResponse)
	if !ok {
		err = fmt.Errorf("expected proposal response, got %T", resp)
		return
	}

	ok, reason = _resp.Accepted, _resp.RejectReason
	return
}
