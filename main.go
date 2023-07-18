package main

import (
	"avax-indexer/db"
	"avax-indexer/rpc"
	"avax-indexer/ws"
	"context"
	"github.com/onrik/ethrpc"
	"golang.org/x/exp/slog"
	"os"
	"os/signal"
)

const (
	defaultRPCAvalanche = "https://api.avax.network/ext/bc/C/rpc"
	defaultWSAvalanche  = "wss://api.avax.network/ext/bc/C/ws"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	rpcHost := os.Getenv("AVAX_RPC")
	if rpcHost == "" {
		rpcHost = defaultRPCAvalanche
	}

	rpcInfura := os.Getenv("AVAX_RPC_INFURA")
	if rpcInfura == "" {
		slog.Error("AVAX_RPC_INFURA env var is required")
		return
	}

	wsHost := os.Getenv("AVAX_WS")
	if wsHost == "" {
		wsHost = defaultWSAvalanche
	}

	dbHost := os.Getenv("MONGODB_URI")
	if dbHost == "" {
		slog.Error("MONGODB_URI env var is required")
		return
	}

	mongoDb, err := db.InitMongoConn(dbHost)
	if err != nil {
		slog.Error("failed to connect to mongo", "error", err)
		return
	}

	repo, err := db.NewBlocksRepo(mongoDb)
	if err != nil {
		slog.Error("failed to initialize blocks repo", "error", err)
		return
	}

	chainRpc := ethrpc.New(rpcHost)
	infuraClient := ethrpc.New(rpcInfura)

	catchUpper := rpc.NewCatchUpper(infuraClient, chainRpc, repo)
	indexer := rpc.NewIndexer(chainRpc, repo)

	if err := catchUpper.CatchUp(); err != nil {
		slog.Error("failed to catch up with blockchain", slog.StringValue(err.Error()))
	}

	c := ws.NewListener(wsHost, indexer)
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
