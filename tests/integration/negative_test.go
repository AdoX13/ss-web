package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMQTTNegativeScenarios(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	mongoC, mosqC, err := SetupTestEnv(ctx)
	require.NoError(t, err)
	defer mongoC.Terminate(ctx)
	defer mosqC.Terminate(ctx)

	host, err := mosqC.Host(ctx)
	require.NoError(t, err)
	port, err := mosqC.MappedPort(ctx, "8883")
	require.NoError(t, err)

	t.Run("Connection fails with invalid cert (no CA verification)", func(t *testing.T) {
		opts := mqtt.NewClientOptions()
		opts.AddBroker(fmt.Sprintf("ssl://%s:%s", host, port.Port()))
		opts.SetClientID("test-client-bad-cert")
		
		// Insecure TLS without the proper CA/client certs
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		})

		client := mqtt.NewClient(opts)
		token := client.Connect()
		token.WaitTimeout(5 * time.Second)

		assert.Error(t, token.Error(), "Broker should reject connection without valid client cert signed by CA")
	})
}
