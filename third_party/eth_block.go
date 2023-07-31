package third_party

import (
	"bytes"
	"github.com/onrik/ethrpc"
	"math/big"
	"unsafe"
)

// ProxyBlockWithTransactions is a proxy for ethrpc.Block
// Sourced from github.com/onrik/ethrpc
type ProxyBlockWithTransactions struct {
	Number           hexInt             `json:"number"`
	Hash             string             `json:"hash"`
	ParentHash       string             `json:"parentHash"`
	Nonce            string             `json:"nonce"`
	Sha3Uncles       string             `json:"sha3Uncles"`
	LogsBloom        string             `json:"logsBloom"`
	TransactionsRoot string             `json:"transactionsRoot"`
	StateRoot        string             `json:"stateRoot"`
	Miner            string             `json:"miner"`
	Difficulty       hexBig             `json:"difficulty"`
	TotalDifficulty  hexBig             `json:"totalDifficulty"`
	ExtraData        string             `json:"extraData"`
	Size             hexInt             `json:"size"`
	GasLimit         hexInt             `json:"gasLimit"`
	GasUsed          hexInt             `json:"gasUsed"`
	Timestamp        hexInt             `json:"timestamp"`
	Uncles           []string           `json:"uncles"`
	Transactions     []ProxyTransaction `json:"transactions"`
}

// ToBlock converts a ProxyBlockWithTransactions to an ethrpc.Block
// Sourced from github.com/onrik/ethrpc
func (proxy *ProxyBlockWithTransactions) ToBlock() ethrpc.Block {
	return *(*ethrpc.Block)(unsafe.Pointer(proxy))
}

// hexInt is a proxy for ethrpc.Block Number
// Sourced from github.com/onrik/ethrpc
type hexInt int

// UnmarshalJSON implements the json.Marshaler interface.
// Sourced from github.com/onrik/ethrpc
func (i *hexInt) UnmarshalJSON(data []byte) error {
	result, err := ParseInt(string(bytes.Trim(data, `"`)))
	*i = hexInt(result)

	return err
}

// hexBig is a proxy for ethrpc.Block Difficulty and TotalDifficulty
// Sourced from github.com/onrik/ethrpc
type hexBig big.Int

// UnmarshalJSON implements the json.Unmarshaler interface.
// Sourced from github.com/onrik/ethrpc
func (i *hexBig) UnmarshalJSON(data []byte) error {
	result, err := ParseBigInt(string(bytes.Trim(data, `"`)))
	*i = hexBig(result)

	return err
}
