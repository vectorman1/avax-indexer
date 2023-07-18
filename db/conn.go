package db

import (
	"avax-indexer/common"
	"context"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slog"
	"time"
)

func InitMongoConn(host common.SecretValue) (*mongo.Database, error) {
	ctx, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()

	opts := options.Client().ApplyURI(string(host))

	slog.Info("connecting to mongo", "host", opts.Hosts)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongo")
	}

	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, errors.Wrap(err, "failed to ping mongo")
	}

	return client.Database("avax-indexer"), nil
}
