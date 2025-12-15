package main

import (
	"flag"
	"log"
	"net/http"
	pClient "service_discovery/pkg/client"
	"service_discovery/pkg/counter"
	"service_discovery/pkg/handler"
	pStore "service_discovery/pkg/peerStore"
	"service_discovery/pkg/service"
	"strings"
	"time"
)

func main() {
	port := flag.String("port", "8010", "port to listen on")
	peers := flag.String("peers", "", "comma separated peers")
	flag.Parse()

	selfID := "localhost:" + *port

	peerStore := pStore.NewPeerStore(selfID)
	peerClient := pClient.NewClient()

	peerCounter := counter.NewCounter()
	peerService := service.NewPeerService(selfID, peerStore, peerClient, peerCounter)
	peerHandler := handler.NewPeerHandler(peerService)
	mux := http.NewServeMux()
	mux.HandleFunc("/nodes/join", peerHandler.Join)
	mux.HandleFunc("/nodes", peerHandler.List)
	mux.HandleFunc("/nodes/heartbeat", peerHandler.Heartbeat)

	mux.HandleFunc("/counter/increment", peerHandler.Increment)
	mux.HandleFunc("/counter/replicate", peerHandler.Replicate)
	mux.HandleFunc("/counter/count", peerHandler.Count)
	log.Println("🚀 Server started on :8080")

	if *peers != "" {
		for _, peer := range strings.Split(*peers, ",") {
			peerService.JoinPeer(peer)
		}
	}

	go peerService.StartHeartbeat()
	go peerService.StartCleanup(5 * time.Second)
	go peerService.StartRetryLoop()

	log.Println("Node running on port", *port)
	log.Fatal(http.ListenAndServe(":"+*port, RequestLogger(mux)))
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf(
			"[API RECEIVED] %s %s",
			r.Method,
			r.URL.Path,
		)
		next.ServeHTTP(w, r)
	})
}
