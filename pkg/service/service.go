package service

import (
	"errors"
	"log"
	"service_discovery/pkg/client"
	"service_discovery/pkg/counter"
	pstore "service_discovery/pkg/peerStore"
	"sync"
	"time"
)

type PeerService struct {
	SelfId  string
	PStore  pstore.IPeerStore
	Client  client.IClient
	Counter counter.IPeerCounter
	Pending map[string][]*PendingEvent
	PMutex  sync.Mutex
}

type PendingEvent struct {
	EventID   string
	Attempt   int
	NextRetry time.Time
}

func NewPeerService(selfId string, p pstore.IPeerStore, cl client.IClient, pCounter counter.IPeerCounter) *PeerService {
	return &PeerService{
		SelfId:  selfId,
		PStore:  p,
		Client:  cl,
		Counter: pCounter,
		Pending: make(map[string][]*PendingEvent),
	}
}

type IPeerService interface {
	JoinPeer(peer string)
	AddPeer(peer string)
	GetPeersList() []string
	Increment(eventID string) error
	GetCounterValue() int64
}

func (s *PeerService) JoinPeer(peer string) {
	peers, err := s.Client.JoinCluster(peer, s.PStore.SelfID())
	if err != nil {
		return
	}

	log.Println("peers are ", peers)
	s.PStore.AddPeer(peer)
	for _, p := range peers {
		s.PStore.AddPeer(p)
	}
}

func (s *PeerService) AddPeer(peer string) {
	s.PStore.AddPeer(peer)
}

func (s *PeerService) GetPeersList() []string {
	return s.PStore.GetPeers()
}

func (s *PeerService) StartHeartbeat() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		for _, peer := range s.PStore.GetPeers() {
			_ = s.Client.Heartbeat(peer, s.PStore.SelfID())
		}
	}
}

func (s *PeerService) StartCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		now := time.Now()
		for peer, last := range s.PStore.SnapshotOfPeers() {
			if now.Sub(last) > 6*time.Second {
				s.PStore.RemovePeer(peer)
			}
		}
	}
}

func (s *PeerService) Increment(eventID string) error {
	applied := s.Counter.Apply(eventID, 1)
	if !applied {
		return errors.New("counter not applied")
	}

	log.Println("Counter applied,sending to peers")

	// Propagate asynchronously to peers
	for _, peer := range s.GetPeersList() {
		go func(p string) {
			go s.sendOrQueue(p, eventID)
		}(peer)
	}

	return nil
}

func (s *PeerService) sendOrQueue(peer, eventID string) {
	if err := s.Client.SendIncrement(peer, s.SelfId, eventID); err != nil {
		s.enqueue(peer, eventID)
	}
}

func (s *PeerService) enqueue(peer, eventID string) {
	s.PMutex.Lock()
	defer s.PMutex.Unlock()

	event := &PendingEvent{
		EventID:   eventID,
		Attempt:   0,
		NextRetry: time.Now(),
	}
	s.Pending[peer] = append(s.Pending[peer], event)
}

func (s *PeerService) GetCounterValue() int64 {

	return s.Counter.Get()
}

func (s *PeerService) StartRetryLoop() {
	go func() {
		for {
			s.syncPending()
			time.Sleep(2 * time.Second)
		}
	}()
}

func (s *PeerService) syncPending() {
	s.PMutex.Lock()
	defer s.PMutex.Unlock()

	now := time.Now()
	for peer, events := range s.Pending {
		var remaining []*PendingEvent

		for _, e := range events {
			if now.Before(e.NextRetry) {
				remaining = append(remaining, e)
				continue
			}

			if err := s.Client.SendIncrement(peer, s.SelfId, e.EventID); err != nil {
				// Failed, schedule next retry with exponential backoff
				e.Attempt++

				delay := time.Millisecond * 100 * (1 << (e.Attempt - 1))
				if delay > time.Second*10 {
					delay = time.Second * 10 // max backoff
				}
				e.NextRetry = time.Now().Add(delay)
				remaining = append(remaining, e)
			}
			// Success - do not add to remaining, effectively removing it
		}

		if len(remaining) == 0 {
			delete(s.Pending, peer)
		} else {
			s.Pending[peer] = remaining
		}
	}
}
