package peerStore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddRemoveGetPeers(t *testing.T) {
	ps := NewPeerStore("self")

	ps.AddPeer("peer1")
	ps.AddPeer("peer2")
	ps.AddPeer("self") // should not add

	peers := ps.GetPeers()
	assert.Contains(t, peers, "peer1")
	assert.Contains(t, peers, "peer2")
	assert.NotContains(t, peers, "self")

	ps.RemovePeer("peer1")
	peers = ps.GetPeers()
	assert.NotContains(t, peers, "peer1")
}

func TestSnapshotOfPeers(t *testing.T) {
	ps := NewPeerStore("self")
	ps.AddPeer("peer1")
	ps.AddPeer("peer2")

	snapshot := ps.SnapshotOfPeers()
	assert.Len(t, snapshot, 2)

	// Ensure snapshot is a copy
	snapshot["peer3"] = time.Now()
	assert.NotContains(t, ps.GetPeers(), "peer3")
}

func TestSelfID(t *testing.T) {
	ps := NewPeerStore("self")
	assert.Equal(t, "self", ps.SelfID())
}
