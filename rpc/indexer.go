package rpc

import (
	"avax-indexer/db"
	"context"
	"github.com/onrik/ethrpc"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
	"time"
)

type Indexer struct {
	rpc  *ethrpc.EthRPC
	repo *db.BlocksRepo
}

func NewIndexer(client *ethrpc.EthRPC, repo *db.BlocksRepo) *Indexer {
	return &Indexer{rpc: client, repo: repo}
}

func (i *Indexer) ProcessBlock(hash string) {
retry:
	time.Sleep(1 * time.Second)
	block, err := i.rpc.EthGetBlockByHash(hash, true)
	if err != nil {
		if e := new(ethrpc.EthError); errors.As(err, e) {
			if e.Code == -32000 {
				slog.Warn("too early; retrying block after 1 second", "hash", hash)
				goto retry
			}
		}
		slog.Error("failed to get block; retrying after 1 second", "hash", hash, "error", err)
		goto retry
	}

	if block == nil {
		slog.Warn("block not found; retrying", "hash", hash)
		goto retry
	}

retryInsert:
	ctx, c := context.WithTimeout(context.Background(), 1*time.Second)
	defer c()

	if err := i.repo.Insert(ctx, block); err != nil {
		slog.Error("failed to insert block; retrying in 1 sec", "hash", block.Hash, "error", err)
		time.Sleep(1 * time.Second)
		goto retryInsert
	}
}
