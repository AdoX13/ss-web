package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMQTTmTLSConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Spin up testcontainers
	mongoC, mosqC, err := SetupTestEnv(ctx)
	require.NoError(t, err)
	defer mongoC.Terminate(ctx)
	defer mosqC.Terminate(ctx)

	host, err := mosqC.Host(ctx)
	require.NoError(t, err)

	port, err := mosqC.MappedPort(ctx, "8883")
	require.NoError(t, err)

	t.Run("Valid mTLS connection succeeds", func(t *testing.T) {
		client, err := NewTestMQTTClient(host, port.Port(), "test-client-valid")
		require.NoError(t, err)
		defer client.Disconnect(250)

		assert.True(t, client.IsConnected())
	})

	t.Run("Plaintext connection fails (rejected by broker)", func(t *testing.T) {
		opts := mqtt.NewClientOptions()
		opts.AddBroker(fmt.Sprintf("tcp://%s:%s", host, port.Port()))
		opts.SetClientID("test-client-invalid")
		// No TLS config provided

		client := mqtt.NewClient(opts)
		token := client.Connect()
		token.WaitTimeout(5 * time.Second)

		assert.Error(t, token.Error(), "Connection should fail without TLS")
	})

	t.Run("Publishing message over mTLS succeeds", func(t *testing.T) {
		client, err := NewTestMQTTClient(host, port.Port(), "test-client-pub")
		require.NoError(t, err)
		defer client.Disconnect(250)

		token := client.Publish("test/topic", 1, false, "hello mtls")
		token.WaitTimeout(2 * time.Second)
		
		assert.NoError(t, token.Error())
	})
}
