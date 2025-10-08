package message

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	ethwallet "github.com/perun-network/perun-eth-backend/wallet"
	stewallet "perun.network/perun-stellar-backend/wallet/types"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
)

// ChannelProposal sends an incoming channel proposal to the websocket client
// and returns its proposal response.
func (c *Connection) ChannelProposal(
	lcp *client.LedgerChannelProposalMsg, myIdx, peerIdx channel.Index,
) (propResp *ProposalResponse, err error) {
	bals, peerBals := MakeBals(lcp.InitBals.Balances, myIdx, peerIdx)
	state := NewChannelState(lcp.InitBals.Assets, lcp.InitBals.Backends, bals, peerBals, false)
	ethAddress, ok := lcp.Participant[EthereumIndex].(*ethwallet.Address)
	if !ok {
		ethAddress = &ethwallet.Address{}
	}
	steAddress, ok := lcp.Participant[SolanaIndex]
	if !ok {
		steAddress = &stewallet.Participant{}
	}
	req := &ChannelProposal{
		ID:             lcp.ProposalID,
		PeerAddressEth: common.Address(*ethAddress),
		PeerAddressSte: steAddress.String(),
		State:          state,
	}
	log.Println("State: ", state)

	resp, err := c.Request(req)
	if err != nil {
		return
	}

	propResp, ok = resp.(*ProposalResponse)
	if !ok {
		err = fmt.Errorf("expected proposal response, got %T", resp)
		return
	}
	return
}
