package handler

import (
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net/http"
	"service_discovery/pkg/service"
)

type PeerHandler struct {
	Service service.IPeerService
}

func NewPeerHandler(s service.IPeerService) *PeerHandler {
	return &PeerHandler{Service: s}
}

type NodeRequestBody struct {
	NodeID string `json:"node_id"`
}

func (ph *PeerHandler) Join(w http.ResponseWriter, r *http.Request) {

	var body NodeRequestBody
	err := json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		log.Println("error in decoding the body", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
	}

	ph.Service.AddPeer(body.NodeID)

	resp := map[string][]string{
		"peers": ph.Service.GetPeersList(),
	}
	log.Println("peers are", resp)
	json.NewEncoder(w).Encode(resp)
}

func (h *PeerHandler) List(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(h.Service.GetPeersList())
}

func (h *PeerHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var body NodeRequestBody
	err := json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		log.Println("error in decoding the body", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
	}

	h.Service.AddPeer(body.NodeID)
	w.WriteHeader(http.StatusOK)
}

func (h *PeerHandler) Increment(w http.ResponseWriter, _ *http.Request) {
	eventID := uuid.NewString()

	log.Println("Received request for increment of counter", eventID)

	h.Service.Increment(eventID)

	w.WriteHeader(http.StatusOK)
}

type ReplicateBody struct {
	EventID string `json:"event_id"`
}

func (h *PeerHandler) Replicate(w http.ResponseWriter, r *http.Request) {
	var body ReplicateBody
	log.Println("received request to replicate counter", body.EventID)
	_ = json.NewDecoder(r.Body).Decode(&body)

	h.Service.Increment(body.EventID)
	w.WriteHeader(http.StatusOK)
}

func (h *PeerHandler) Count(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(map[string]int64{
		"count": h.Service.GetCounterValue(),
	})
}
