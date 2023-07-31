package db

// Block represents a block in the blockchain, annotated for MongoDB
type Block struct {
	Number           int           `bson:"number"`
	Hash             string        `bson:"hash"`
	ParentHash       string        `bson:"parent_hash"`
	Nonce            string        `bson:"nonce"`
	Sha3Uncles       string        `bson:"sha_3_uncles"`
	LogsBloom        string        `bson:"logs_bloom"`
	TransactionsRoot string        `bson:"transactions_root"`
	StateRoot        string        `bson:"state_root"`
	Miner            string        `bson:"miner"`
	Difficulty       string        `bson:"difficulty"`
	TotalDifficulty  string        `bson:"total_difficulty"`
	ExtraData        string        `bson:"extra_data"`
	Size             int           `bson:"size"`
	GasLimit         int           `bson:"gas_limit"`
	GasUsed          int           `bson:"gas_used"`
	Timestamp        int           `bson:"timestamp"`
	Uncles           []string      `bson:"uncles"`
	Transactions     []Transaction `bson:"transactions"`
}

// Transaction represents a transaction in the blockchain, annotated for MongoDB
type Transaction struct {
	Hash             string `bson:"hash"`
	Nonce            int    `bson:"nonce"`
	BlockHash        string `bson:"block_hash"`
	BlockNumber      *int   `bson:"block_number"`
	TransactionIndex *int   `bson:"transaction_index"`
	From             string `bson:"from"`
	To               string `bson:"to"`
	Value            string `bson:"value"`
	Gas              int    `bson:"gas"`
	GasPrice         string `bson:"gas_price"`
	Input            string `bson:"input"`
}
