package rpc

import (
	"avax-indexer/db"
	"avax-indexer/model"
	"avax-indexer/third_party"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/onrik/ethrpc"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"time"
)

const blockRange = 9900

type CatchUpper struct {
	chainRpc  *ethrpc.EthRPC
	infuraRpc *ethrpc.EthRPC
	http      *http.Client
	repo      *db.BlocksRepo
}

type Req struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
	Jsonrpc string        `json:"jsonrpc"`
}

func NewCatchUpper(infuraRpc *ethrpc.EthRPC, chainRpc *ethrpc.EthRPC, repo *db.BlocksRepo) *CatchUpper {
	return &CatchUpper{
		chainRpc:  chainRpc,
		infuraRpc: infuraRpc,
		repo:      repo,
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *CatchUpper) CatchUp() error {
	currBlock, err := c.chainRpc.EthBlockNumber()
	if err != nil {
		return errors.Wrap(err, "failed to get current block number")
	}

	req, err := c.prepareRequestForPreviousBlocks(currBlock)
	if err != nil {
		return errors.Wrap(err, "failed to prepare request")
	}
	slog.Info("prepared request for catching up with blocks", "count", blockRange)

	slog.Info("fetching blocks", "count", blockRange)
	response, err := c.http.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}
	defer response.Body.Close()

	slog.Info("decoding blocks", "count", blockRange)
	var res []*model.Params[third_party.ProxyBlockWithTransactions]
	if err := json.NewDecoder(response.Body).Decode(&res); err != nil {
		if err == io.EOF {
			return nil
		}
		return errors.Wrap(err, "failed to decode response body")
	}
	slog.Info("received blocks", "count", len(res))

	blocks := make([]*ethrpc.Block, len(res))
	for i, blk := range res {
		b := blk.Result.ToBlock()
		blocks[i] = &b
	}

	if err := c.repo.UpsertMany(context.Background(), blocks); err != nil {
		return errors.Wrap(err, "failed to upsert catching up blocks")
	}

	return nil
}

func (c *CatchUpper) prepareRequestForPreviousBlocks(currBlock int) (*http.Request, error) {
	reqs := make([]Req, blockRange)
	for i := 0; i < blockRange; i++ {
		reqs[i] = Req{
			Method:  "eth_getBlockByNumber",
			Params:  []interface{}{fmt.Sprintf("0x%x", currBlock), true},
			Id:      i,
			Jsonrpc: "2.0",
		}
		currBlock--
	}

	b, err := json.Marshal(reqs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest(http.MethodPost, c.infuraRpc.URL(), bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}
