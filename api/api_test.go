package api_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/OmGuptaIND/api"
	"github.com/stretchr/testify/assert"
)

func TestApiServer(t *testing.T) {
	apiServer := api.NewApiServer(api.ApiServerOptions{
		Port: 3000,
	})

	assert.NotNil(t, apiServer)

	go apiServer.Start()

	defer apiServer.Close()

	time.Sleep(3 * time.Second)

	resp, err := http.Get("http://localhost:3000/ping")

	assert.Nil(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	assert.Nil(t, err)

	assert.Equal(t, "pong", string(body))
}
