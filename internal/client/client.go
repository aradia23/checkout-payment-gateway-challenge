package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	baseURL string
	client  *http.Client
}

type AutherizaionRequest struct {
	CardNumber string `json:"card_number"`
	ExpiryDate string `json:"expiry_date"`
	Currency   string `json:"currency"`
	Amount     int    `json:"amount"`
	Cvv        string `json:"cvv"`
}

type AutherizaionResponse struct {
	Authorizationed   string `json:"authorizationed"`
	AuthorizationCode string `json:"authorization_code"`
}

//go:generate mockery --name ClientInterface
type ClientInterface interface {
	Authorize(req AutherizaionRequest) (AutherizaionResponse, error)
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *Client) Authorize(req AutherizaionRequest) (AutherizaionResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return AutherizaionResponse{}, fmt.Errorf("marshalling bank request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/payments", bytes.NewReader(payload))
	if err != nil {
		return AutherizaionResponse{}, fmt.Errorf("building bank request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return AutherizaionResponse{}, fmt.Errorf("calling bank: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// e.g. 400 (malformed request on our side - shouldn't happen once we
		// validate) or 503 (bank/simulator unavailable, per imposters config).
		if resp.StatusCode == http.StatusBadRequest {
			return AutherizaionResponse{}, fmt.Errorf("bank returned status %d: malformed request", resp.StatusCode)
		}
		if resp.StatusCode == http.StatusServiceUnavailable {
			return AutherizaionResponse{}, fmt.Errorf("bank returned status %d: service unavailable", resp.StatusCode)
		}
	}

	var authResp AutherizaionResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return AutherizaionResponse{}, fmt.Errorf("decoding bank response: %w", err)
	}

	return authResp, nil
}
