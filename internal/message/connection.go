package message

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"perun.network/go-perun/log"
	pkgsync "polycry.pt/poly-go/sync"
)

const (
	// MaxNumRequests is the maximum number of requests a client is allowed to
	// open up in parallel.
	MaxNumRequests = 16
	// The interval in which the websocket will send a ping to the client. Has
	// to be shorter than pongTimeout.
	pingInterval = 20 * time.Second
	// Maximum time the client has to respond to a ping with a pong.
	pongTimeout = 60 * time.Second
)

// Connection represents a websocket connection to a client.
type Connection struct {
	conn             *websocket.Conn
	readMu           sync.Mutex
	writeMu          sync.Mutex
	closer           pkgsync.Closer
	onClose          func()
	requestCounter   uint64
	requestCounterMu sync.Mutex
	responseHandlers *messageHandlerMap
}

// NewConnection creates a new connection from a websocket connection.
func NewConnection(conn *websocket.Conn) *Connection {
	return &Connection{
		conn:             conn,
		requestCounter:   0,
		responseHandlers: newMessageHandlerMap(),
	}
}

func (c *Connection) Read() (msg Message, err error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	mt, b, err := c.conn.ReadMessage()
	if err != nil {
		return
	}

	if mt == websocket.TextMessage {
		var obj JSONObject
		err = obj.UnmarshalJSON(b)
		if err == nil {
			msg = obj.Message
		}
	} else {
		err = fmt.Errorf("invalid message type: %v", mt)
	}

	return
}

func (c *Connection) Write(msg Message) (err error) {
	obj := &JSONObject{Message: msg}
	b, err := obj.MarshalJSON()
	if err != nil {
		return
	}

	err = c.write(websocket.TextMessage, b)
	return
}

func (c *Connection) write(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(messageType, data)
}

// CloseWithError closes the connection with an error message.
func (c *Connection) CloseWithError(error error) error {
	err := c.Write(NewError(error))
	if err != nil {
		return err
	}
	err = c.Close()
	return err
}

// SetOnCloseHandler sets the closing handler which is called once the
// connection is closed. Does not have to be thread-safe.
func (c *Connection) SetOnCloseHandler(onClose func()) {
	c.onClose = onClose
}

// A Handler handles requests sent by the websocket client over the connection.
type Handler interface {
	// HandleRequest handles incoming requests from the websocket client and
	// answers with a response matching the request ID. Has to be thread-safe.
	HandleRequest(req Request)
}

// Handle reads messages from the connection and calls the HandleRequest method
func (c *Connection) Handle(h Handler) (err error) {
	err = c.keepAlive()
	if err != nil {
		return errors.Wrap(err, "connection keep alive: %v")
	}

	reserve := make(chan struct{}, MaxNumRequests)
	defer close(reserve)
	for {
		var msg Message
		msg, err = c.Read()
		if err != nil {
			return
		}
		switch msg := msg.(type) {
		case *Response:
			reqID := msg.ID
			_resp, ok := c.responseHandlers.get(reqID)
			if !ok {
				log.Error(fmt.Sprintf("connection %v: response handler %v not found", c, reqID))
			}
			if msg.Message == nil {
				log.Error("received nil response")
				break
			}
			_resp <- msg.Message.Message
		case *Request:
			if msg.Message == nil {
				log.Error("received nil request")
				break
			}
			// Ensure that the number of open requests stays below the constant
			// MaxNumRequests. Otherwise, respond with an error message.
			go func() {
				select {
				case reserve <- struct{}{}:
					defer func() {
						<-reserve
					}()
				default:
					resp := NewResponse(msg.ID, &Error{Err: "Exceeding maximum number of allowed open requests"})
					err := c.Write(resp)
					if err != nil {
						log.Error(errors.Wrap(err, "sending max requests error"))
					}
					return
				}
				h.HandleRequest(*msg)
			}()
		}
	}
}

// Close closes the connection.
func (c *Connection) Close() error {
	if err := c.closer.Close(); err != nil {
		return err
	}
	if c.onClose != nil {
		c.onClose()
	}
	return c.conn.Close()
}

// keepAlive sets the pong handler and starts sending pings periodically.
func (c *Connection) keepAlive() error {
	// We initially set the read deadline to the pongTimeout.
	err := c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	if err != nil {
		return errors.Wrap(err, "setting read deadline")
	}
	c.conn.SetPongHandler(func(appData string) error {
		// We renew the read deadline with every pong we receive.
		err := c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
		if err != nil {
			log.Errorf("setting read deadline failed: %v", err)
		}
		return nil
	})

	// Start sending pings in the defined pingInterval.
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := c.write(websocket.PingMessage, nil)
				if err != nil {
					log.Errorf("sending ping failed: %v", err)
					return
				}
			case <-c.closer.Closed():
				return
			}
		}
	}()

	return nil
}
