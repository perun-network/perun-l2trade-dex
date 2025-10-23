package websocket

import (
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/perun-network/perun-dex-websocket/internal/client"
	"github.com/sirupsen/logrus"
	"perun.network/go-perun/log"
	plogrus "perun.network/go-perun/log/logrus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Run runs the node by handling requests to /connect.
func Run(config Config) {
	http.Handle("/", http.FileServer(http.Dir("./web")))

	http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		connect(w, r, config.ClientConfig)
	})
	log.Infof("Websocket running on %s", config.WSAddress)

	// Add order book streaming endpoint
	http.HandleFunc("/ws/orderbook", ServeOrderBookStream)

	if config.TLSCertificate != "" && config.TLSPrivKey != "" {
		log.Fatal(http.ListenAndServeTLS(config.WSAddress, config.TLSCertificate, config.TLSPrivKey, nil))
	} else {
		log.Fatal(http.ListenAndServe(config.WSAddress, nil))
	}
}

// connect is started whenever a client connects to the entrypoint of the
// websocket and starts the Perun client.
func connect(w http.ResponseWriter, r *http.Request, cfg client.Config) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalf("upgrade:", err)
		return
	}
	startClient(conn, cfg)
}

func init() {
	// Set the logging output file
	logger := logrus.New()

	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	log.Set(plogrus.FromLogrus(logger))
}
