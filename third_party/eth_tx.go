package third_party

type ProxyTransaction struct {
	Hash             string  `json:"hash"`
	Nonce            hexInt  `json:"nonce"`
	BlockHash        string  `json:"blockHash"`
	BlockNumber      *hexInt `json:"blockNumber"`
	TransactionIndex *hexInt `json:"transactionIndex"`
	From             string  `json:"from"`
	To               string  `json:"to"`
	Value            hexBig  `json:"value"`
	Gas              hexInt  `json:"gas"`
	GasPrice         hexBig  `json:"gasPrice"`
	Input            string  `json:"input"`
}
