package client

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}
}

type IClient interface {
	JoinCluster(peerId, selfId string) ([]string, error)
	Heartbeat(peer, selfID string) error
	SendIncrement(peer, selfId, eventId string) error
}

type Payload struct {
	NodeId string `json:"node_id"`
}

type JoinClusterResponse struct {
	Peers []string `json:"peers"`
}

func (c *Client) JoinCluster(peerId, selfId string) ([]string, error) {
	payload := Payload{
		NodeId: selfId,
	}

	var result []string
	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		log.Println("error in marshalling the payload bytes", err)
		return result, err
	}

	url := "http://" + peerId + "/nodes/join"

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		log.Println("error in forming the request", err)
		return result, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Println("error in sending the client request", err)
		return result, err
	}
	defer resp.Body.Close()

	var Response JoinClusterResponse

	err = json.NewDecoder(resp.Body).Decode(&Response)

	if err != nil {
		log.Println("error in decoding the response ", err)
	}
	result = Response.Peers

	return result, nil

}

func (c *Client) Heartbeat(peer, selfID string) error {
	payload := Payload{
		NodeId: selfID,
	}

	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		log.Println("error in marshalling the payload bytes", err)
		return err
	}

	url := "http://" + peer + "/nodes/heartbeat"

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		log.Println("error in forming the request", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Println("error in sending the client request", err)
		return err
	}
	defer resp.Body.Close()

	return nil
}

type SendIncrementPayload struct {
	NodeId  string `json:"node_id"`
	EventId string `json:"event_id"`
}

func (c *Client) SendIncrement(peer, selfId, eventId string) error {
	payload := SendIncrementPayload{
		NodeId:  selfId,
		EventId: eventId,
	}

	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		log.Println("error in marshalling the payload bytes", err)
		return err
	}

	url := "http://" + peer + "/counter/replicate"

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		log.Println("error in forming the request", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Println("error in sending the client request", err)
		return err
	}
	defer resp.Body.Close()

	return nil
}
