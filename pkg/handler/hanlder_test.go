package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"service_discovery/mocks/service_discovery/pkg/service"
)

func TestJoinHandler(t *testing.T) {
	mockService := &service.MockIPeerService{}

	// Setup expectations for the mock
	mockService.On("AddPeer", "peer1").Return()
	mockService.On("GetPeersList").Return([]string{"peer1", "peer2"})

	handler := NewPeerHandler(mockService)

	// Prepare HTTP request
	req := httptest.NewRequest(http.MethodPost, "/join", strings.NewReader(`{"node_id":"peer1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Join(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "peer1")
	assert.Contains(t, string(body), "peer2")

	// Assert that all expectations were met
	mockService.AssertExpectations(t)
}

func TestListHandler(t *testing.T) {
	mockService := &service.MockIPeerService{}
	handler := NewPeerHandler(mockService)

	mockService.On("GetPeersList").Return([]string{"peer1", "peer2"})

	req := httptest.NewRequest(http.MethodGet, "/list", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	var respBody []string
	json.NewDecoder(resp.Body).Decode(&respBody)

	assert.Equal(t, []string{"peer1", "peer2"}, respBody)
	mockService.AssertCalled(t, "GetPeersList")
}

func TestHeartbeatHandler(t *testing.T) {
	mockService := &service.MockIPeerService{}
	handler := NewPeerHandler(mockService)

	mockService.On("AddPeer", "peer1").Return()

	body := map[string]string{"node_id": "peer1"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handler.Heartbeat(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	mockService.AssertCalled(t, "AddPeer", "peer1")
}

func TestIncrementHandler(t *testing.T) {
	mockService := &service.MockIPeerService{}
	handler := NewPeerHandler(mockService)

	mockService.On("Increment", mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/increment", nil)
	w := httptest.NewRecorder()

	handler.Increment(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	mockService.AssertCalled(t, "Increment", mock.Anything)
}

func TestCountHandler(t *testing.T) {
	mockService := &service.MockIPeerService{}
	handler := NewPeerHandler(mockService)

	mockService.On("GetCounterValue").Return(int64(42))

	req := httptest.NewRequest(http.MethodGet, "/count", nil)
	w := httptest.NewRecorder()

	handler.Count(w, req)

	var respBody map[string]int64
	json.NewDecoder(w.Body).Decode(&respBody)

	assert.Equal(t, int64(42), respBody["count"])
	mockService.AssertCalled(t, "GetCounterValue")
}
