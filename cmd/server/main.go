package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/perun-network/perun-dex-websocket/internal/client"
	"github.com/perun-network/perun-dex-websocket/internal/websocket"
)

var (
	runCmd = flag.NewFlagSet("run", flag.ExitOnError)
)

func main() {
	args := os.Args[1:]
	run(args)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}

func run(args []string) {
	var (
		addr               = runCmd.String("addr", "127.0.0.1:8080", "http service address")
		runEthChainsFile   = runCmd.String("ethChains", "chains.yaml", "Ethereum Chains config file")
		runSolChainsFile   = runCmd.String("solChains", "chains_solana.yaml", "Solana Chains config file")
		cert               = runCmd.String("cert", "", "TLS certificate")
		certKey            = runCmd.String("certKey", "", "Private key for the TLS certificate")
		defaultTimeout     = runCmd.Duration("defaultTimeout", 1*time.Minute, "Default timeout")
		handleTimeout      = runCmd.Duration("handleTimeout", 5*time.Minute, "Timeout for handling proposals")
		fundTimeout        = runCmd.Duration("fundTimeout", 10*time.Minute, "Timeout for funding channels")
		settleTimeout      = runCmd.Duration("settleTimeout", 10*time.Minute, "Timeout for settling channels")
		runTxFinalityDepth = runCmd.Uint64("finalityDepth", 1, "Number of confirmations required to confirm a blockchain transaction")
		predefinedGasLimit = runCmd.Bool("predefinedGasLimit", false, "Predefined gas limit for all transactions")
		horizonURL         = runCmd.String("horizonURL", "http://localhost:8000", "URL of the Stellar Horizon server")
	)
	err := runCmd.Parse(args)
	if err != nil {
		log.Fatalf("parsing run flags: %v", err)
		return
	}

	ethChainsConfig, err := websocket.ParseEthereumChainsConfig(*runEthChainsFile)
	if err != nil {
		log.Fatalf("parsing chain config file: %v", err)
	}

	solChainsConfig, err := websocket.ParseSolanaChainsConfig(*runSolChainsFile)
	if err != nil {
		log.Fatalf("parsing chain config file: %v", err)
	}

	cfg := websocket.Config{
		WSAddress:      *addr,
		TLSCertificate: *cert,
		TLSPrivKey:     *certKey,
		ClientConfig: client.Config{
			EthChains: ethChainsConfig.ChainMap(),
			SolChains: solChainsConfig.AssetMap(),
			GasLimits: websocket.GasLimits(*predefinedGasLimit),
			Timeouts: client.Timeouts{
				DefaultTimeout: *defaultTimeout,
				HandleTimeout:  *handleTimeout,
				FundTimeout:    *fundTimeout,
				SettleTimeout:  *settleTimeout,
			},
			TxFinalityDepth: *runTxFinalityDepth,
			HorizonURL:      *horizonURL,
		},
	}
	fmt.Println("cfg: ", solChainsConfig.AssetMap())
	websocket.Run(cfg)
}
