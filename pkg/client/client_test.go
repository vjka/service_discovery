package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinCluster(t *testing.T) {
	// Create a fake server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{"peer1", "peer2"})
	}))
	defer server.Close()

	c := &Client{httpClient: server.Client()}
	peers, err := c.JoinCluster(server.Listener.Addr().String(), "self")
	assert.NoError(t, err)
	assert.Equal(t, []string{"peer1", "peer2"}, peers)
}

func TestHeartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{httpClient: server.Client()}
	err := c.Heartbeat(server.Listener.Addr().String(), "self")
	assert.NoError(t, err)
}

func TestSendIncrement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{httpClient: server.Client()}
	err := c.SendIncrement(server.Listener.Addr().String(), "self", "event1")
	assert.NoError(t, err)
}
