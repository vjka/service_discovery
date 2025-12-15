package service

import (
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"service_discovery/mocks/service_discovery/pkg/client"
	"service_discovery/mocks/service_discovery/pkg/peerStore"
	"service_discovery/pkg/counter"
	"sync"
	"testing"
	"time"
)

func TestJoinPeer_AddsPeers(t *testing.T) {
	mockClient := &client.MockIClient{}
	mockStore := &peerStore.MockIPeerStore{}

	// Setup expectations
	mockStore.On("SelfID").Return("self1")
	mockStore.On("AddPeer", "peer1").Return()
	mockStore.On("AddPeer", "peer2").Return()

	mockClient.On("JoinCluster", "peer1", "self1").Return([]string{"peer2"}, nil)

	service := NewPeerService("self1", mockStore, mockClient, counter.NewCounter())

	service.JoinPeer("peer1")

	mockStore.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestAddPeer(t *testing.T) {
	mockStore := &peerStore.MockIPeerStore{}
	c := counter.NewCounter()
	mockClient := &client.MockIClient{}
	service := NewPeerService("self", mockStore, mockClient, c)

	mockStore.On("AddPeer", "peer1").Return()

	service.AddPeer("peer1")

	mockStore.AssertCalled(t, "AddPeer", "peer1")
}

func TestGetPeersList(t *testing.T) {
	mockStore := &peerStore.MockIPeerStore{}
	c := counter.NewCounter()
	mockClient := &client.MockIClient{}
	service := NewPeerService("self", mockStore, mockClient, c)

	mockStore.On("GetPeers").Return([]string{"peer1", "peer2"})

	peers := service.GetPeersList()

	assert.Equal(t, []string{"peer1", "peer2"}, peers)
	mockStore.AssertCalled(t, "GetPeers")
}

func TestIncrement_Applied(t *testing.T) {
	mockClient := &client.MockIClient{}
	mockStore := &peerStore.MockIPeerStore{}

	// Setup peers
	mockStore.On("GetPeers").Return([]string{"peer1"})
	mockStore.On("SelfID").Return("self")

	// Expect SendIncrement to be called
	called := make(chan bool, 1)
	mockClient.On("SendIncrement", "peer1", "self", "event1").Return(nil).Run(func(args mock.Arguments) {
		called <- true
	})

	c := counter.NewCounter()
	svc := NewPeerService("self", mockStore, mockClient, c)

	// Call Increment
	_ = svc.Increment("event1")

	// Wait for SendIncrement to be called
	select {
	case <-called:
		// success
	case <-time.After(time.Second):
		t.Fatal("SendIncrement was not called")
	}

	mockClient.AssertExpectations(t)
}

func TestIncrement_AlreadyApplied(t *testing.T) {
	mockClient := &client.MockIClient{}
	mockStore := &peerStore.MockIPeerStore{}
	c := counter.NewCounter()
	service := NewPeerService("self", mockStore, mockClient, c)

	service.Counter.Apply("event1", 1)

	err := service.Increment("event1")
	assert.Error(t, err)
	assert.Equal(t, "counter not applied", err.Error())
}

func TestGetCounterValue(t *testing.T) {
	mockClient := &client.MockIClient{}
	mockStore := &peerStore.MockIPeerStore{}
	c := counter.NewCounter()
	c.Apply("event1", 1)

	service := NewPeerService("self", mockStore, mockClient, c)

	val := service.GetCounterValue()
	assert.Equal(t, int64(1), val)
}

func TestSendOrQueue_Failure(t *testing.T) {
	mockClient := &client.MockIClient{}
	mockStore := &peerStore.MockIPeerStore{}
	c := counter.NewCounter()
	service := NewPeerService("self", mockStore, mockClient, c)

	mockClient.On("SendIncrement", "peer1", "self", "event1").Return(errors.New("fail"))

	service.sendOrQueue("peer1", "event1")

	service.PMutex.Lock()
	defer service.PMutex.Unlock()
	assert.Len(t, service.Pending["peer1"], 1)
	assert.Equal(t, "event1", service.Pending["peer1"][0].EventID)
}

func TestConcurrentIncrements(t *testing.T) {
	mockStore := &peerStore.MockIPeerStore{}
	mockClient := &client.MockIClient{}

	mockStore.On("GetPeers").Return([]string{})
	mockStore.On("SelfID").Return("node1")

	c := counter.NewCounter()
	svc := NewPeerService("node1", mockStore, mockClient, c)

	// Run multiple concurrent increments
	var wg sync.WaitGroup
	concurrent := 100
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.Increment(uuid.NewString())
		}()
	}
	wg.Wait()

	if c.Get() != int64(concurrent) {
		t.Fatalf("expected counter %d, got %d", concurrent, c.Get())
	}
}

func TestIncrementPropagation(t *testing.T) {
	mockStore := &peerStore.MockIPeerStore{}
	mockClient := &client.MockIClient{}

	mockStore.On("GetPeers").Return([]string{"node2"})
	mockStore.On("SelfID").Return("node1")

	c := counter.NewCounter()
	svc := NewPeerService("node1", mockStore, mockClient, c)

	// Capture call to SendIncrement
	called := make(chan bool, 1)
	mockClient.On("SendIncrement", "node2", "node1", "event1").Return(nil).Run(func(args mock.Arguments) {
		called <- true
	})

	_ = svc.Increment("event1")

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("SendIncrement was not called for propagation")
	}

	mockClient.AssertExpectations(t)
}

func TestDeduplicationAndRetry(t *testing.T) {
	mockStore := &peerStore.MockIPeerStore{}
	mockClient := &client.MockIClient{}

	mockStore.On("GetPeers").Return([]string{"peer1"})
	mockStore.On("SelfID").Return("self")

	c := counter.NewCounter()
	svc := NewPeerService("self", mockStore, mockClient, c)

	// First attempt succeeds
	mockClient.On("SendIncrement", "peer1", "self", "event1").Return(errors.New("network error"))

	_ = svc.Increment("event1")

	// Start retry loop
	svc.StartRetryLoop()

	// Wait for retry attempt
	time.Sleep(250 * time.Millisecond)

	// Counter should only be incremented once
	if c.Get() != 1 {
		t.Fatalf("counter incremented more than once, got %d", c.Get())
	}
}

func TestClusterRebalance(t *testing.T) {
	// Create mock PeerStore
	mockStore := &peerStore.MockIPeerStore{}
	mockStore.On("SnapshotOfPeers").Return(map[string]time.Time{
		"node1": time.Now(),
		"node2": time.Now().Add(-10 * time.Second), // should be removed
	})
	mockStore.On("RemovePeer", "node2").Return()

	// Create dummy client and counter
	mockClient := &client.MockIClient{}
	counter := counter.NewCounter()

	// Create PeerService with mockStore
	svc := NewPeerService("self", mockStore, mockClient, counter)

	// Start cleanup with short interval for testing
	go svc.StartCleanup(10 * time.Millisecond)

	// Wait briefly to allow cleanup goroutine to run
	time.Sleep(50 * time.Millisecond)

	// Assert RemovePeer was called for node2
	mockStore.AssertCalled(t, "RemovePeer", "node2")
	mockStore.AssertNotCalled(t, "RemovePeer", "node1")
}
