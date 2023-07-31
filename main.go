package main

import (
	"avax-indexer/common"
	"avax-indexer/db"
	"avax-indexer/rpc"
	"avax-indexer/ws"
	"context"
	"github.com/onrik/ethrpc"
	"golang.org/x/exp/slog"
	"os"
	"os/signal"
	"strconv"
)

const (
	defaultRPCAvalanche = "https://api.avax.network/ext/bc/C/rpc"
	defaultWSAvalanche  = "wss://api.avax.network/ext/bc/C/ws"
)

type env struct {
	rpcHost    string
	wsHost     string
	rpcInfura  common.SecretValue
	dbHost     common.SecretValue
	blocksNum  int64
	avgDocSize int64
}

var cfg env

func init() {
	// Load env vars
	rpcHost := os.Getenv("AVAX_RPC")
	if rpcHost == "" {
		slog.Info("AVAX_RPC env var is not set; using default for mainnet", "host", defaultRPCAvalanche)
		rpcHost = defaultRPCAvalanche
	}
	wsHost := os.Getenv("AVAX_WS")
	if wsHost == "" {
		slog.Info("AVAX_WS env var is not set; using default for mainnet", "host", defaultWSAvalanche)
		wsHost = defaultWSAvalanche
	}
	rpcInfura := common.SecretValue(os.Getenv("AVAX_RPC_INFURA"))
	if rpcInfura == "" {
		slog.Error("AVAX_RPC_INFURA env var is required")
		return
	}
	dbHost := common.SecretValue(os.Getenv("MONGODB_URI"))
	if dbHost == "" {
		slog.Error("MONGODB_URI env var is required")
		return
	}
	blocksNumStr := os.Getenv("BLOCKS")
	if blocksNumStr == "" {
		slog.Info("BLOCKS env var is not set; using default", "blocks", 10000)
		blocksNumStr = "10000"
	}
	blocksNum, err := strconv.Atoi(blocksNumStr)
	if err != nil {
		slog.Error("failed to parse BLOCKS env var", "error", err)
		return
	}
	avgDocSizeStr := os.Getenv("AVG_DOC_SIZE")
	if avgDocSizeStr == "" {
		slog.Info("AVG_DOC_SIZE env var is not set; using default", "size", 50)
		avgDocSizeStr = "50"
	}
	avgDocSize, err := strconv.Atoi(avgDocSizeStr)
	if err != nil {
		slog.Error("failed to parse AVG_DOC_SIZE env var", "error", err)
		return
	}

	cfg = env{
		rpcHost:    rpcHost,
		wsHost:     wsHost,
		rpcInfura:  rpcInfura,
		dbHost:     dbHost,
		blocksNum:  int64(blocksNum),
		avgDocSize: int64(avgDocSize),
	}
}

func main() {
	// Setup interrupt handler
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	mongoDb, err := db.InitMongoConn(cfg.dbHost)
	if err != nil {
		slog.Error("failed to connect to mongo", "error", err)
		return
	}

	repo, err := db.NewMongoBlocksRepo(mongoDb, cfg.blocksNum, cfg.avgDocSize)
	if err != nil {
		slog.Error("failed to initialize blocks repo", "error", err)
		return
	}

	// Initialize ETH clients
	// We are using the main Avalanche C-Chain RPC endpoint for
	// the chain client, and Infura for the bulk requests for catching up
	// with missed blocks
	chainClient := ethrpc.New(cfg.rpcHost)
	infuraClient := ethrpc.New(string(cfg.rpcInfura))

	// Initialize services
	catchUpper := rpc.NewCatchUpper(infuraClient, chainClient, repo, cfg.blocksNum)
	indexer := rpc.NewIndexer(chainClient, repo)

	// Catch up with missed blocks
	if err := catchUpper.CatchUp(); err != nil {
		slog.Error("failed to catch up with blockchain", "error", err.Error())
		os.Exit(1)
	}

	c := ws.NewListener(cfg.wsHost, indexer)
	if err := c.Subscribe(); err != nil {
		slog.Error("failed to subscribe to newHeads", "error", err)
	}

	for {
		select {
		case <-c.Done():
			return
		case <-interrupt:
			if err := c.GraceClose(); err != nil {
				slog.Error("failed to gracefully close ws connection", "error", err)
			}
			slog.Info("disconnecting from mongo")
			if err := mongoDb.Client().Disconnect(context.Background()); err != nil {
				slog.Error("failed to disconnect from mongo", "error", err)
			}
		}
	}
}
