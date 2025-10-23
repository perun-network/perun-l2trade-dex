package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/perun-network/perun-dex-websocket/internal/client"
	"github.com/perun-network/perun-dex-websocket/internal/deploy/ethereum"
	"github.com/perun-network/perun-dex-websocket/internal/websocket"
)

const ()

var (
	runCmd = flag.NewFlagSet("run", flag.ExitOnError)
)

func main() {
	args := os.Args[1:]
	run(args)
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

	// Deploy Ethereum contracts.
	fmt.Println("Deploying Ethereum contracts...")
	adj, ah := ethereum.DeployContracts(ethChainsConfig.Chains[0].NodeURL, ethChainsConfig.Chains[0].ChainID.Uint64(), ethChainsConfig.Chains[0].DeployerSK)
	fmt.Println("Deployed Ethereum contracts:")
	fmt.Println("  Adjudicator:", adj.Hex())
	fmt.Println("  Asset Holder:", ah.Hex())

	ethChainsConfig.Chains[0].Adjudicator = adj
	ethChainsConfig.Chains[0].Assets[0].AssetHolder = ah

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
		},
	}
	fmt.Println("cfg: ", solChainsConfig.AssetMap())
	websocket.Run(cfg)
}
