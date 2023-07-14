package main

import (
	"avax-indexer/db"
	"avax-indexer/rpc"
	"avax-indexer/ws"
	"context"
	"github.com/onrik/ethrpc"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"os"
	"os/signal"
)

func init() {
	viper.SetConfigFile("config.yml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	rpcHost := viper.GetString("rpc.host")
	rpcInfura := viper.GetString("rpc.infura")
	wsHost := viper.GetString("ws.host")
	dbHost := viper.GetString("db.host")

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

	c := ws.NewListener(wsHost, indexer)

	if err := c.Subscribe(); err != nil {
		slog.Error("failed to subscribe to newHeads", "error", err)
	}

	go func() {
		if err := catchUpper.CatchUp(); err != nil {
			slog.Error("failed to catch up with blockchain", "error", err)
		}
	}()

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
