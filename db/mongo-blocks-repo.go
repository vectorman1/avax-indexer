package db

import (
	"context"
	"github.com/onrik/ethrpc"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

const blocksCollection = "blocks"
const blocksRange = 10000
const avgBlockSizeBytes = 50 * 1000 // 50 kb

// MongoBlocksRepo is a repository for blocks
type MongoBlocksRepo struct {
	db *mongo.Database
}

// NewMongoBlocksRepo initializes a new blocks repository
// If the blocks collection does not exist, it will be created
// and indexes will be created
func NewMongoBlocksRepo(db *mongo.Database, num int64, docSize int64) (*MongoBlocksRepo, error) {
	colls, err := db.ListCollectionNames(context.Background(), bson.M{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list collection names")
	}

	if !slices.Contains(colls, blocksCollection) {
		maxSize := num * docSize * 1000
		slog.Info("creating blocks capped collection", "cap", num, "bytes_size", maxSize)
		// create capped blocks collection
		opts := options.CreateCollection()
		opts.SetCapped(true)
		opts.SetMaxDocuments(num)
		opts.SetSizeInBytes(maxSize)
		if err := db.CreateCollection(context.Background(), blocksCollection, opts); err != nil {
			slog.Error("failed to create capped blocks collection", "error", err)
			return nil, errors.Wrap(err, "failed to create capped blocks collection")
		}

		// create indexes
		idx := []mongo.IndexModel{
			{
				Keys: bson.D{{
					Key:   "number",
					Value: -1,
				}},
			},
			{
				Keys: bson.D{{
					Key:   "timestamp",
					Value: -1,
				}},
			},
			{
				Keys: bson.D{{
					Key:   "hash",
					Value: -1,
				}},
			},
			{
				Keys: bson.D{
					{
						Key:   "transactions.from",
						Value: -1,
					},
					{
						Key:   "transactions.value",
						Value: -1,
					},
				},
			},
			{
				Keys: bson.D{
					{
						Key:   "transactions.to",
						Value: -1,
					},
					{
						Key:   "transactions.value",
						Value: -1,
					},
				},
			},
			{
				Keys: bson.D{
					{
						Key:   "transactions.hash",
						Value: -1,
					},
				},
			},
			{
				Keys: bson.D{
					{
						Key:   "transactions.block_number",
						Value: -1,
					},
					{
						Key:   "transactions.transaction_index",
						Value: -1,
					},
				},
			},
		}
		slog.Info("creating indexes", "count", len(idx))
		if _, err := db.Collection(blocksCollection).Indexes().CreateMany(context.Background(), idx); err != nil {
			return nil, errors.Wrap(err, "failed to create indexes")
		}
	}

	return &MongoBlocksRepo{db: db}, nil
}

// Insert inserts a block into the database
func (r *MongoBlocksRepo) Insert(ctx context.Context, block *ethrpc.Block) error {
	m := Block{}.FromResponse(block)
	opts := options.Update().
		SetUpsert(true)
	f := bson.M{
		"hash": m.Hash,
	}
	u := bson.M{
		"$set": m,
	}

	_, err := r.db.Collection(blocksCollection).UpdateOne(ctx, f, u, opts)
	return err
}

// UpsertMany inserts or updates many blocks into the database
func (r *MongoBlocksRepo) UpsertMany(ctx context.Context, blocks []*ethrpc.Block) error {
	models := make([]mongo.WriteModel, 0)
	for i := len(blocks) - 1; i >= 0; i-- {
		b := blocks[i]
		m := Block{}.FromResponse(b)

		upd := mongo.NewUpdateOneModel().
			SetUpsert(true).
			SetFilter(bson.M{
				"hash": m.Hash,
			}).
			SetUpdate(bson.M{
				"$set": m,
			})

		models = append(models, upd)
	}

	_, err := r.db.Collection(blocksCollection).
		BulkWrite(ctx, models)
	if err != nil {
		return errors.Wrap(err, "failed to bulk upsert blocks")
	}

	return nil
}

// LastHead returns the last block number in the database
func (r *MongoBlocksRepo) LastHead(ctx context.Context) (int64, error) {
	agg := []bson.M{
		{
			"$sort": bson.M{
				"number": -1,
			},
		},
		{
			"$limit": 1,
		},
		{
			"$project": bson.M{
				"number": 1,
			},
		},
	}

	cur, err := r.db.Collection(blocksCollection).
		Aggregate(ctx, agg)
	if err != nil {
		return 0, errors.Wrap(err, "failed to aggregate latest block number")
	}
	defer cur.Close(ctx)

	var res struct {
		Number int64 `bson:"number"`
	}
	if !cur.Next(ctx) {
		return 0, nil
	}
	if err := cur.Decode(&res); err != nil {
		return 0, errors.Wrap(err, "failed to decode latest block number")
	}
	if cur.Err() != nil {
		return 0, errors.Wrap(cur.Err(), "failed to iterate latest block number")
	}

	return res.Number, nil
}
