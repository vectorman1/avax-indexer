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

type BlocksRepo struct {
	db *mongo.Database
}

func NewBlocksRepo(db *mongo.Database) (*BlocksRepo, error) {
	colls, err := db.ListCollectionNames(context.Background(), bson.M{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list collection names")
	}

	if !slices.Contains(colls, blocksCollection) {
		slog.Info("creating blocks capped collection")
		// create capped blocks collection
		opts := options.CreateCollection()
		opts.SetCapped(true)
		opts.SetMaxDocuments(blocksRange)
		opts.SetSizeInBytes(avgBlockSizeBytes * blocksRange)
		if err := db.CreateCollection(context.Background(), blocksCollection, opts); err != nil {
			slog.Error("failed to create capped blocks collection", "error", err)
			return nil, errors.Wrap(err, "failed to create capped blocks collection")
		}

		// create indexes
		slog.Info("creating indexes")
		if _, err := db.Collection(blocksCollection).Indexes().CreateMany(context.Background(), []mongo.IndexModel{
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
		}); err != nil {
			return nil, errors.Wrap(err, "failed to create indexes")
		}
	}

	return &BlocksRepo{db: db}, nil
}

func (r *BlocksRepo) Insert(ctx context.Context, block *ethrpc.Block) error {
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

func (r *BlocksRepo) UpsertMany(ctx context.Context, blocks []*ethrpc.Block) error {
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

func (r *BlocksRepo) LastHead(ctx context.Context) (int, error) {
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
		Number int `bson:"number"`
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
