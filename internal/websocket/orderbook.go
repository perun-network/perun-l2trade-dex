package websocket

import (
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/perun-network/perun-dex-websocket/internal/client"
	"github.com/perun-network/perun-dex-websocket/internal/message"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/log"
)

var streamUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeOrderBookStream handles /ws/orderbook?channel=<hex> for streaming.
func ServeOrderBookStream(w http.ResponseWriter, r *http.Request) {
	chHex := r.URL.Query().Get("channel")
	if chHex == "" {
		http.Error(w, "missing channel parameter", http.StatusBadRequest)
		return
	}

	var chID channel.ID
	bs, err := hex.DecodeString(chHex)
	if err != nil || len(bs) != len(chID) {
		http.Error(w, "invalid channel id", http.StatusBadRequest)
		return
	}
	copy(chID[:], bs)

	conn, err := streamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("upgrade failed:", err)
		return
	}
	defer conn.Close()

	book := client.OrderBookEngine.GetOrCreateBook(chID)

	// Send initial snapshot
	snap := book.Snapshot()
	snapFrame := message.JSONObject{Message: &snap}
	if err := conn.WriteJSON(snapFrame); err != nil {
		return
	}

	// Subscribe to deltas
	deltaCh := make(chan []byte, 64)
	book.Subscribe(deltaCh)
	defer book.Unsubscribe(deltaCh)

	// Send deltas
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case data := <-deltaCh:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			// Send ping to keep connection alive
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
