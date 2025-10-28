package client

import (
	"crypto/ecdsa"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/perun-network/perun-dex-websocket/internal/message"
	ethwallet "github.com/perun-network/perun-eth-backend/wallet"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
	"perun.network/go-perun/client"
	"perun.network/go-perun/log"
)

type Client struct {
	addr        common.Address                      // L2 address
	addrs       map[wallet.BackendID]wallet.Address // L1 addresses
	wireAddrs   map[wallet.BackendID]wire.Address
	conn        *message.Connection
	perunClient *client.Client
	adjudicator *multi.Adjudicator

	solChains SolanaChainMap
	ethChains EthereumChainMap
	Timeouts  Timeouts

	chMtx    sync.RWMutex // Protects the channels.
	channels map[channel.ID]*client.Channel

	reg *Registry
}

// NewClient creates a new client.
func NewClient(
	conn *message.Connection,
	l2sk *ecdsa.PrivateKey,
	eaddr common.Address,
	saddr string,
	cfg Config,
	reg *Registry,
) (*Client, error) {
	walletAddrs, perunClient, adjudicator, wireAddrs, err := newPerunClient(conn, l2sk, eaddr, saddr, cfg)
	if err != nil {
		return nil, err
	}
	l2AddrEth := walletAddrs[ethwallet.BackendID].(*ethwallet.Address)
	l2Addr := (*common.Address)(l2AddrEth)
	c := &Client{
		addr:        *l2Addr,
		addrs:       walletAddrs,
		wireAddrs:   wireAddrs,
		conn:        conn,
		perunClient: perunClient,
		adjudicator: adjudicator,
		channels:    make(map[channel.ID]*client.Channel),
		solChains:   cfg.SolChains,
		ethChains:   cfg.EthChains,
		Timeouts:    cfg.Timeouts,
		reg:         reg,
	}
	return c, nil
}

// Run starts the client.
func (c *Client) Run() {
	defer c.perunClient.Close()
	defer c.shutdown()

	cond := sync.NewCond(&sync.Mutex{})

	go func() {
		err := c.handleProposals()
		c.log("proposal handler closed", err)
		cond.Signal()
	}()

	go func() {
		handler := &requestHandler{c}
		err := c.conn.Handle(handler)
		c.log("message handler closed:", err)
		cond.Signal()
	}()
	c.log("Started")

	err := c.conn.Write(&message.Initialized{L2Address: c.addr})
	if err != nil {
		c.log("sending initialized", err)
	}

	cond.L.Lock()
	cond.Wait()
	cond.L.Unlock()
}

// CloseWithError closes the client with an error.
func (c *Client) CloseWithError(err error) {
	log.Error(err)
	c.shutdown()
}

// check all Assets in the config for validity
func (c *Client) checkAssets(assets []message.Asset) error {
	for _, a := range assets {
		var supported bool
		// Check if the asset is supported by the solana chain.
		asset, ok := a.(*message.SolanaAsset)
		if ok {
			_, ok := c.solChains[0].Assets[asset.Code()]
			if !ok {
				return fmt.Errorf("unsupported Asset with ID=%v, %v", asset.Mint, c.solChains[0].Assets)
			}
		} else {
			// Check if the asset is supported by one of the ethereum chains.
			asset, ok := a.(*message.EthereumAsset)
			if ok {
				chn, ok := c.ethChains[asset.ChainID.MapKey()]
				if !ok {
					return fmt.Errorf("unsupported chain with ID=%v", asset.ChainID)
				}
				for _, ca := range chn.Assets {
					if ca.AssetHolder == asset.AssetHolder {
						supported = true
						break
					}
				}
				if !supported {
					return fmt.Errorf("unsupported asset %v", a)
				}
			}
		}
	}
	return nil
}

func (c *Client) addChannel(ch *client.Channel) {
	c.chMtx.Lock()
	defer c.chMtx.Unlock()
	c.channels[ch.ID()] = ch
}

func (c *Client) getChannel(id channel.ID) (*client.Channel, bool) {
	c.chMtx.RLock()
	defer c.chMtx.RUnlock()
	ch, ok := c.channels[id]
	return ch, ok
}

func (c *Client) removeChannel(id channel.ID) {
	c.chMtx.Lock()
	defer c.chMtx.Unlock()
	log.Println("Removing channel: ", id)
	if c.channels[id] != nil {
		delete(c.channels, id)
	}
}

func (c *Client) shutdown() {
	var err error
	err = c.conn.Close()
	if err != nil {
		c.log(err)
	}

	err = c.perunClient.Close()
	if err != nil {
		c.log(err)
	}
}

func (c *Client) log(v ...interface{}) {
	log.Printf("Client %v: %s", c.addr, fmt.Sprint(v...))
}
