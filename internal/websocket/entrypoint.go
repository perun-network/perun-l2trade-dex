package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/perun-network/perun-dex-websocket/internal/client"
	"github.com/perun-network/perun-dex-websocket/internal/message"

	"perun.network/go-perun/log"
)

var clients = client.NewRegistry()

// startClient registers a client to the clients map with its sent address and
// starts a new client if there isn't already a client registered/running with
// the same address.
func startClient(conn *websocket.Conn, cfg client.Config) {
	mconn := message.NewConnection(conn)

	msg, err := mconn.Read()
	if err != nil {
		log.Print("read message:", err)
		return
	}

	initMsg, err := message.Initialize(msg)
	if err != nil {
		log.Warnf("expected initialization message, got %T", msg)
		return
	}

	// Handle CC initialization message
	if crossInitMsg, ok := initMsg.(*message.CrossContractInitialize); ok {
		log.Println("Got cross-contract init message")
		// Create L1 and L2 Addresses for client
		solClientAddr := crossInitMsg.SolClientAddress
		ethClientAddr := crossInitMsg.EthClientAddress
		c, err := clients.Register(ethClientAddr, solClientAddr, mconn, cfg)
		if err != nil {
			if err := mconn.CloseWithError(err); err != nil {
				log.Error(err)
			}
			return
		}

		mconn.SetOnCloseHandler(func() {
			clients.Remove(ethClientAddr.String())
			clients.Remove(solClientAddr)
		})

		go c.Run()
	}
}
