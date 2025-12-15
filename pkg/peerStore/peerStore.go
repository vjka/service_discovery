package peerStore

import (
	"sync"
	"time"
)

type PeerStore struct {
	ID    string
	Mutex sync.RWMutex
	Peers map[string]time.Time
}

func NewPeerStore(peerId string) *PeerStore {
	return &PeerStore{
		ID:    peerId,
		Peers: make(map[string]time.Time),
	}
}

type IPeerStore interface {
	AddPeer(peerId string)
	RemovePeer(peerId string)
	GetPeers() []string
	SelfID() string
	SnapshotOfPeers() map[string]time.Time
}

func (ps *PeerStore) AddPeer(peerId string) {
	if peerId == ps.ID {
		return
	}
	ps.Mutex.Lock()
	defer ps.Mutex.Unlock()
	ps.Peers[peerId] = time.Now()
}

func (ps *PeerStore) RemovePeer(peer string) {
	ps.Mutex.Lock()
	defer ps.Mutex.Unlock()
	delete(ps.Peers, peer)
}

func (ps *PeerStore) GetPeers() []string {
	ps.Mutex.RLock()
	defer ps.Mutex.RUnlock()

	peers := make([]string, 0, len(ps.Peers))
	for p := range ps.Peers {
		peers = append(peers, p)
	}
	return peers
}
func (ps *PeerStore) SelfID() string {
	return ps.ID
}

func (ps *PeerStore) SnapshotOfPeers() map[string]time.Time {
	ps.Mutex.RLock()
	defer ps.Mutex.RUnlock()

	copy := make(map[string]time.Time)
	for k, v := range ps.Peers {
		copy[k] = v
	}
	return copy
}
