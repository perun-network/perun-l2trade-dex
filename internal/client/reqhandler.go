package client

import (
	"math"
	mathrand "math/rand"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/perun-network/perun-eth-backend/bindings/peruntoken"
	ethwallet "github.com/perun-network/perun-eth-backend/wallet"
	solchannel "github.com/perun-network/perun-solana-backend/channel"
	solclient "github.com/perun-network/perun-solana-backend/client"
	solwallet "github.com/perun-network/perun-solana-backend/wallet"

	"github.com/pkg/errors"

	"github.com/perun-network/perun-dex-websocket/internal/message"

	"perun.network/go-perun/client"
	"perun.network/go-perun/log"
)

type requestHandler struct {
	*Client
}

// HandleRequest maps requests to the corresponding handler.
func (h *requestHandler) HandleRequest(req message.Request) {
	var respMsg message.Message

	// Route order book messages first via the global engine you already have.
	if resp, handled := h.HandleOrderBookMessage(req.Message.Message); handled {
		respMsg = resp
	} else {
		switch reqMsg := req.Message.Message.(type) {
		case *message.GetChannelInfo:
			respMsg = h.handleGetChannelInfo(reqMsg)
		case *message.GetSignedState:
			respMsg = h.handleGetSignedState(reqMsg)
		case *message.SignedState:
			respMsg = h.handleSendSignedState(reqMsg)
		case *message.GetChains:
			respMsg = h.handleGetChains()
		case *message.GetAssets:
			respMsg = h.handleGetAssets(reqMsg)
		case *message.GetDecimals:
			respMsg = h.handleGetDecimals(reqMsg)
		case *message.GetTimeout:
			respMsg = h.handleGetTimeout(reqMsg)
		case *message.GetQuote:
			respMsg = h.handleGetQuote(reqMsg)
		case *message.GetFunds:
			respMsg = h.handleGetFunds(reqMsg)
		case *message.GetBalance:
			respMsg = h.handleGetBalance(reqMsg)
		case *message.GetHubBalance:
			respMsg = h.handleGetHubBalance(reqMsg)
		default:
			respMsg = h.handleChannelAction(reqMsg)
		}
	}

	resp := message.NewResponse(req.ID, respMsg)
	if err := h.conn.Write(resp); err != nil {
		h.log("sending response", err)
	}

}

func (c *Client) handleGetChannelInfo(msg *message.GetChannelInfo) message.Message {
	ch, ok := c.getChannel(msg.ID)
	if !ok {
		return &message.Error{Err: "channel not found"}
	}

	myIdx, peerIdx := ch.Idx(), 1-ch.Idx()
	bals, peerBals := message.MakeBals(ch.State().Balances, myIdx, peerIdx)
	peerAddrEth := common.Address{}
	peerAddrSol := ""
	if _, ok := ch.Params().Parts[peerIdx][message.EthereumIndex]; ok {
		peerAddrEthWallet := ch.Params().Parts[peerIdx][message.EthereumIndex].(*ethwallet.Address)
		peerAddrEth = *(*common.Address)(peerAddrEthWallet)
	}
	if _, ok := ch.Params().Parts[peerIdx][message.SolanaIndex]; ok {
		peerAddrSolPart := ch.Params().Parts[peerIdx][message.SolanaIndex].(*solwallet.Participant)
		peerAddrSol = peerAddrSolPart.SolanaAddress.String()
	}

	return &message.ChannelInfo{
		PeerAddressEth: peerAddrEth,
		PeerAddressSol: peerAddrSol,
		State:          message.NewChannelState(ch.State().Assets, ch.State().Backends, bals, peerBals, ch.State().IsFinal),
	}
}

func (c *Client) handleGetSignedState(msg *message.GetSignedState) message.Message {
	ch, ok := c.getChannel(msg.ID)
	if !ok {
		return &message.Error{Err: "channel not found"}
	}
	tch := client.NewTransparentChannel(ch)
	signedState := message.SignedState(tch.SignedState())
	return &signedState
}

func (c *Client) handleSendSignedState(msg *message.SignedState) message.Message {
	if msg.State == nil || msg.Sigs == nil || msg.Params == nil {
		return message.NewError(errors.New("handleSendSignedState: unknown timeout type"))
	} else {
		return &message.Success{}
	}
}

func (c *Client) handleGetChains() message.Message {
	// We declare a non-nil but zero-length slice because we want to encode it
	// with JSON such that empty slices encode to [] and not to null.
	var chains []message.ChainInfo

	for _, c := range c.ethChains {
		mc := message.ChainInfo{
			Name: c.Name,
			ID:   c.ChainID,
		}
		chains = append(chains, mc)
	}
	for _, c := range c.solChains {
		mc := message.ChainInfo{
			Name: c.Name,
			ID:   c.ChainID,
		}
		chains = append(chains, mc)
	}
	return &message.GetChainsResponse{Chains: chains}
}

func (c *Client) handleGetAssets(msg *message.GetAssets) message.Message {
	// We declare a non-nil but zero-length slice because we want to encode it
	// with JSON such that empty slices encode to [] and not to null.
	var ethAsets []message.EthereumAssetConfig
	for _, id := range msg.ChainIDs {
		chn, ok := c.ethChains[id.MapKey()]
		if !ok {
			continue
		}
		cAssets := message.EthereumAssetMapToArray(chn.Assets)
		ethAsets = append(ethAsets, cAssets...)
	}
	var solAsets []message.SolanaAssetConfig
	for range msg.ChainIDs {
		cAssets := message.SolanaAssetMapToArray(c.solChains[0].Assets)
		solAsets = append(solAsets, cAssets...)
	}

	return &message.GetAssetsResponse{SolanaAssets: solAsets, EthereumAssets: ethAsets}
}

func (c *Client) handleGetDecimals(msg *message.GetDecimals) message.Message {
	switch msg.Asset.AssetType() {
	case "Solana":
		dec, err := c.getSolanaAssetDecimals(*(msg.Asset.(*message.SolanaAsset)))
		if err != nil {
			log.Errorf("handleGetDecimals: %v", err)
			return &message.Error{Err: "could not fetch the decimals for the asset"}
		}

		return &message.GetDecimalsResponse{Decimals: dec}
	case "Ethereum":
		dec, err := c.getEthereumAssetDecimals(*(msg.Asset.(*message.EthereumAsset)))
		if err != nil {
			log.Errorf("handleGetDecimals: %v", err)
			return &message.Error{Err: "could not fetch the decimals for the asset"}
		}

		return &message.GetDecimalsResponse{Decimals: dec}
	}

	return &message.Error{Err: "Asset not found"}
}

func (c *Client) getSolanaAssetDecimals(asset message.SolanaAsset) (uint8, error) {
	return 9, nil
}

func (c *Client) getEthereumAssetDecimals(asset message.EthereumAsset) (uint8, error) {
	chain, ok := c.ethChains[asset.ChainID.MapKey()]
	if !ok {
		return 0, errors.Errorf("unknown chain %v", asset.ChainID)
	}

	ethClient, err := getEthClient(chain.NodeURL)
	if err != nil {
		return 0, errors.Wrap(err, "creating EthClient")
	}

	var assetCfg message.EthereumAssetConfig
	var assetFound bool

	for _, a := range chain.Assets {
		if asset.AssetHolder == a.AssetHolder {
			assetCfg = a
			assetFound = true
		}
	}
	if !assetFound {
		return 0, errors.New("unknown asset")
	}

	switch assetCfg.Type {
	case message.AssetTypeETH:
		dec := math.Log10(params.Ether)
		return uint8(dec), nil
	case message.AssetTypeERC20:
		token, err := peruntoken.NewPeruntoken(assetCfg.Address, ethClient)
		if err != nil {
			return 0, err
		}
		return token.Decimals(&bind.CallOpts{})
	default:
		return 0, errors.New("unknown asset type")
	}
}

func (c *Client) handleGetBalance(asset *message.GetBalance) message.Message {
	part, ok := c.addrs[message.SolanaIndex].(*solwallet.Participant)
	if !ok {
		return &message.Error{Err: "participant not found"}
	}
	fromAddress := part.SolanaAddress
	rng := mathrand.New(mathrand.NewSource(0))
	account, err := solwallet.NewRandomAccountWithAddress(rng, fromAddress)
	if err != nil {
		return &message.Error{Err: "account not found"}
	}

	sender := NewWebSocketSender(c.conn, &fromAddress, rpc.New(c.solChains[0].NodeURL))
	tc := solclient.NewSignerConfig(
		nil,
		nil,
		account,
		sender,
		c.solChains[0].NodeURL,
	)
	// tc.SetKeyPair(layer2Account)
	cb := solclient.NewContractBackend(*tc, solchannel.BackendID)
	add, err := message.StringToSolanaPublicKey(asset.Asset.Mint)
	if err != nil {
		return &message.Error{Err: "address not valid"}
	}
	bal, err := cb.GetBalance(add)
	if err != nil {
		return &message.Error{Err: "could not get balance"}
	}
	return &message.GetBalanceResponse{Balance: bal}
}

func (c *Client) handleGetTimeout(msg *message.GetTimeout) message.Message {
	var timeout float64

	switch msg.TimeoutType {
	case message.DefaultTimeout:
		timeout = c.Timeouts.DefaultTimeout.Seconds()
	case message.HandleTimeout:
		timeout = c.Timeouts.HandleTimeout.Seconds()
	case message.FundTimeout:
		timeout = c.Timeouts.FundTimeout.Seconds()
	case message.SettleTimeout:
		timeout = c.Timeouts.SettleTimeout.Seconds()
	default:
		return message.NewError(errors.New("unknown timeout type"))
	}

	return &message.GetTimeoutResponse{Timeout: int64(timeout)}
}

func (c *Client) handleGetQuote(msg *message.GetQuote) message.Message {
	hub, ok := c.reg.Get(msg.Hub)
	if !ok {
		return &message.Error{Err: "hub not found"}
	}
	rspChan := make(chan message.Message)
	// Request the hub for the quote.
	go func() {
		rsp, err := hub.conn.Request(msg)
		if err != nil {
			log.Errorf("sending GetQuote request: %v", err)
			rspChan <- &message.Error{Err: "requesting quote failed"}
		}

		gqRsp, ok := rsp.(*message.GetQuoteResponse)
		if !ok {
			log.Errorf("expected getQuote response, got %T", rsp)
			rspChan <- &message.Error{Err: "hub did not answered request properly"}
		}
		rspChan <- gqRsp
	}()

	// Ensure that we receive the answer in time.
	select {
	case rsp := <-rspChan:
		return rsp
	case <-time.After(c.Timeouts.DefaultTimeout):
		return &message.Error{Err: "hub did not answered in time"}
	}
}

func (c *Client) handleGetFunds(msg *message.GetFunds) message.Message {
	//TODO: implement get funds
	return &message.GetFundsResponse{Success: false}
}

func (c *Client) handleChannelAction(msg message.Message) message.Message {
	var err error
	switch msg := msg.(type) {
	case *message.OpenChannel:
		err = c.handleOpenChannel(msg)
	case *message.UpdateChannel:
		err = c.handleUpdateChannel(msg)
	case *message.CloseChannel:
		err = c.handleCloseChannel(msg)
	}

	if err != nil {
		return message.NewError(err)
	} else {
		return &message.Success{}
	}
}

func (c *Client) handleGetHubBalance(msg *message.GetHubBalance) message.Message {
	hub, ok := c.reg.Get(msg.Hub)
	if !ok {
		return &message.Error{Err: "hub not found"}
	}
	rspChan := make(chan message.Message)
	// Request the hub for the quote.
	go func() {
		rsp, err := hub.conn.Request(msg)
		if err != nil {
			log.Errorf("sending GetHubBalance request: %v", err)
			rspChan <- &message.Error{Err: "requesting quote failed"}
		}

		gqRsp, ok := rsp.(*message.GetBalanceResponse)
		if !ok {
			log.Errorf("expected getQuote response, got %T", rsp)
			rspChan <- &message.Error{Err: "hub did not answered request properly"}
		}
		rspChan <- gqRsp
	}()

	// Ensure that we receive the answer in time.
	select {
	case rsp := <-rspChan:
		return rsp
	case <-time.After(c.Timeouts.DefaultTimeout):
		return &message.Error{Err: "hub did not answered in time"}
	}
}
