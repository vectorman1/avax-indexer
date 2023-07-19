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

const maxBlockRange = 9900
const infuraDailyError = `daily request count exceeded, request rate limited`

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

	storedHead, err := c.repo.LastHead(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to get last head")
	}

	if currBlock == storedHead {
		return nil
	}

	req, err := c.prepareRequestForPreviousBlocks(currBlock, storedHead)
	if err != nil {
		return errors.Wrap(err, "failed to prepare request")
	}

	slog.Info("sending request for missing blocks")
	rs, err := c.http.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}
	defer rs.Body.Close()

	if rs.StatusCode != http.StatusOK {
		if rs.StatusCode == http.StatusTooManyRequests {
			var infErr model.InfuraError
			if err := json.NewDecoder(rs.Body).Decode(&infErr); err != nil {
				return errors.Wrap(err, "failed to decode infura error")
			}

			slog.Warn("got Infura 429; waiting", "backoff_seconds", infErr.Data.Rate.BackoffSeconds)
			time.Sleep(time.Duration(infErr.Data.Rate.BackoffSeconds) * time.Second)

			return c.CatchUp()
		}

		return fmt.Errorf("got status code %d", rs.StatusCode)
	}

	slog.Info("decoding blocks")
	var res []*model.Params[third_party.ProxyBlockWithTransactions]
	if err := json.NewDecoder(rs.Body).Decode(&res); err != nil {
		if err == io.EOF {
			return nil
		}
		return errors.Wrap(err, "failed to decode response body")
	}

	blocks := make([]*ethrpc.Block, len(res))
	for i, blk := range res {
		b := blk.Result.ToBlock()
		blocks[i] = &b
	}

	if err := c.repo.UpsertMany(context.Background(), blocks); err != nil {
		return errors.Wrap(err, "failed to upsert catching up blocks")
	}
	slog.Info("saved blocks", "count", len(blocks))

	slog.Info("checking if we need to continue catching up")
	latestHead, err := c.chainRpc.EthBlockNumber()
	if err != nil {
		return errors.Wrap(err, "failed to get latest head")
	}

	if latestHead > currBlock {
		slog.Info("need to continue catching up", "stored_head", currBlock, "latest_head", latestHead)
		return c.CatchUp()
	}

	return nil
}

func (c *CatchUpper) prepareRequestForPreviousBlocks(currHead int, storedHead int) (*http.Request, error) {
	blocksToFetch := maxBlockRange
	if storedHead != 0 {
		missing := currHead - storedHead
		if missing < maxBlockRange {
			blocksToFetch = missing
		}
	}

	blockNum := currHead
	reqs := make([]Req, 0)
	for i := 0; i < blocksToFetch; i++ {
		reqs = append(reqs, Req{
			Method:  "eth_getBlockByNumber",
			Params:  []interface{}{fmt.Sprintf("0x%x", blockNum), true},
			Id:      i,
			Jsonrpc: "2.0",
		})
		blockNum--
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

	slog.Info("prepared request for blocks", "count", blocksToFetch, "from", blockNum, "to", currHead)
	return req, nil
}
