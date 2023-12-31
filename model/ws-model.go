package model

// WsResponse represents a response from Infura's websocket subscriptions
type WsResponse[T any] struct {
	JsonRpc string    `json:"jsonrpc"`
	Method  string    `json:"method"`
	Params  Params[T] `json:"params"`
}

// Params represents the params of a WsResponse
type Params[T any] struct {
	Result T `json:"result"`
}

// NewHead represents the result of a newHeads subscription
type NewHead struct {
	ParentHash       string `json:"parentHash"`
	Sha3Uncles       string `json:"sha3Uncles"`
	Miner            string `json:"miner"`
	StateRoot        string `json:"stateRoot"`
	TransactionsRoot string `json:"transactionsRoot"`
	ReceiptsRoot     string `json:"receiptsRoot"`
	LogsBloom        string `json:"logsBloom"`
	Difficulty       string `json:"difficulty"`
	Number           string `json:"number"`
	GasLimit         string `json:"gasLimit"`
	GasUsed          string `json:"gasUsed"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extraData"`
	MixHash          string `json:"mixHash"`
	Nonce            string `json:"nonce"`
	ExtDataHash      string `json:"extDataHash"`
	BaseFeePerGas    string `json:"baseFeePerGas"`
	ExtDataGasUsed   string `json:"extDataGasUsed"`
	BlockGasCost     string `json:"blockGasCost"`
	Hash             string `json:"hash"`
}
