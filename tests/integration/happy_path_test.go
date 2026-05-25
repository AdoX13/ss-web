package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoHappyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	mongoC, _, err := SetupTestEnv(ctx)
	require.NoError(t, err)
	defer mongoC.Terminate(ctx)

	host, err := mongoC.Host(ctx)
	require.NoError(t, err)
	port, err := mongoC.MappedPort(ctx, "27017")
	require.NoError(t, err)

	uri := "mongodb://admin:supersecret@" + host + ":" + port.Port() + "/"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	err = client.Ping(ctx, nil)
	require.NoError(t, err, "Should be able to ping authenticated Mongo instance")

	db := client.Database("mqtt-streaming-server")
	col := db.Collection("test_docs")

	t.Run("Insert and retrieve document", func(t *testing.T) {
		doc := bson.M{"device_id": "test-device", "status": "processed"}
		res, err := col.InsertOne(ctx, doc)
		require.NoError(t, err)
		assert.NotNil(t, res.InsertedID)

		var result bson.M
		err = col.FindOne(ctx, bson.M{"device_id": "test-device"}).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "processed", result["status"])
	})
}
