package db

import "github.com/onrik/ethrpc"

// FromResponse maps an ethrpc.Block to a domain Block
func (Block) FromResponse(block *ethrpc.Block) *Block {
	mapTx := func(tx []ethrpc.Transaction) []Transaction {
		if tx == nil {
			return nil
		}
		result := make([]Transaction, len(tx))
		for i, t := range tx {
			result[i] = *Transaction{}.FromResponse(&t)
		}
		return result
	}

	return &Block{
		Number:           block.Number,
		Hash:             block.Hash,
		ParentHash:       block.ParentHash,
		Nonce:            block.Nonce,
		Sha3Uncles:       block.Sha3Uncles,
		LogsBloom:        block.LogsBloom,
		TransactionsRoot: block.TransactionsRoot,
		StateRoot:        block.StateRoot,
		Miner:            block.Miner,
		Difficulty:       block.Difficulty.String(),
		TotalDifficulty:  block.TotalDifficulty.String(),
		ExtraData:        block.ExtraData,
		Size:             block.Size,
		GasLimit:         block.GasLimit,
		GasUsed:          block.GasUsed,
		Timestamp:        block.Timestamp,
		Uncles:           block.Uncles,
		Transactions:     mapTx(block.Transactions),
	}
}

// FromResponse maps an ethrpc.Transaction to a domain Transaction
func (Transaction) FromResponse(tx *ethrpc.Transaction) *Transaction {
	return &Transaction{
		Hash:             tx.Hash,
		Nonce:            tx.Nonce,
		BlockHash:        tx.BlockHash,
		BlockNumber:      tx.BlockNumber,
		TransactionIndex: tx.TransactionIndex,
		From:             tx.From,
		To:               tx.To,
		Value:            tx.Value.String(),
		Gas:              tx.Gas,
		GasPrice:         tx.GasPrice.String(),
		Input:            tx.Input,
	}
}
