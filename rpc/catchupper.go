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

// CatchUpper is a service that catches up missing blocks
type CatchUpper struct {
	chainRpc  *ethrpc.EthRPC
	infuraRpc *ethrpc.EthRPC
	http      *http.Client
	repo      *db.MongoBlocksRepo
	blocksNum int64
}

// Req is used for bulk requests to Infura
type Req struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int64         `json:"id"`
	Jsonrpc string        `json:"jsonrpc"`
}

// NewCatchUpper initializes a new CatchUpper service
func NewCatchUpper(infuraRpc *ethrpc.EthRPC, chainRpc *ethrpc.EthRPC, repo *db.MongoBlocksRepo, blocksNum int64) *CatchUpper {
	return &CatchUpper{
		chainRpc:  chainRpc,
		infuraRpc: infuraRpc,
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
		repo:      repo,
		blocksNum: blocksNum,
	}
}

// CatchUp brings the stored blocks up to date with the current head
// It will fetch up to 90%*(10000 or the configured amount) blocks,
// store them in an ordered fashion and then recursively call itself
// until the current head is reached
func (c *CatchUpper) CatchUp() error {
	currBlock, err := c.chainRpc.EthBlockNumber()
	if err != nil {
		return errors.Wrap(err, "failed to get current block number")
	}

	storedHead, err := c.repo.LastHead(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to get last head")
	}

	if int64(currBlock) == storedHead {
		return nil
	}

	req, err := c.prepareRequestForPreviousBlocks(int64(currBlock), storedHead)
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

// prepareRequestForPreviousBlocks prepares a request for the previous blocks
// It will fetch up to 90%*(10000 or the configured amount) blocks
// If the stored head is 0, it will fetch the max amount of blocks
// If the stored head is not 0, it will fetch the difference between the current head and the stored head
func (c *CatchUpper) prepareRequestForPreviousBlocks(currHead int64, storedHead int64) (*http.Request, error) {
	blocksToFetch := int64(0.9 * float32(c.blocksNum))
	if storedHead != 0 {
		missing := currHead - storedHead
		if missing < blocksToFetch {
			blocksToFetch = missing
		}
	}

	blockNum := currHead
	reqs := make([]Req, 0)
	for i := int64(0); i < blocksToFetch; i++ {
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
