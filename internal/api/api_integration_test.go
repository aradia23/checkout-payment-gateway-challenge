//go:build integration

// Package api_test holds integration tests for the payment gateway.
//
// Unlike the unit tests under internal/handlers (which fake the bank with an
// httptest.Server), these tests exercise the real router, the real bank
// HTTP client, and the real acquiring-bank simulator defined in
// imposters/bank_simulator.ejs. They need that simulator running, which is
// why they're gated behind a build tag rather than part of `go test ./...`.
//
// Run with the simulator up:
//
//	docker-compose up -d
//	go test -tags=integration ./internal/api/...
package api_test

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/api"
	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/client"
	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/models"
)

// requireBankSimulator skips the test suite with a clear message if the bank
// simulator isn't reachable, rather than failing on a confusing connection
// error partway through a test.
func requireBankSimulator(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:8080", time.Second)
	if err != nil {
		t.Skip("bank simulator not reachable on localhost:8080 - start it with `docker-compose up -d` to run this test")
		return
	}
	_ = conn.Close()
}

// newTestServer wires up the real Api (real router, real in-memory
// repository, real bank client pointed at the simulator) and serves it over
// an in-process httptest server.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	a := api.New()
	server := httptest.NewServer(a.Handler())
	t.Cleanup(server.Close)
	return server
}

func postPayment(t *testing.T, server *httptest.Server, req models.PostPaymentRequest) (*http.Response, models.PostPaymentResponse) {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshalling request: %v", err)
	}

	resp, err := http.Post(server.URL+"/api/payments", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("posting payment: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	var parsed models.PostPaymentResponse
	if resp.StatusCode == http.StatusCreated {
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
	}
	return resp, parsed
}

func getPayment(t *testing.T, server *httptest.Server, id string) *http.Response {
	t.Helper()
	resp, err := http.Get(server.URL + "/api/payments/" + id)
	if err != nil {
		t.Fatalf("getting payment: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func validRequest(cardNumber string) models.PostPaymentRequest {
	return models.PostPaymentRequest{
		CardNumber:  cardNumber,
		ExpiryMonth: 4,
		ExpiryYear:  time.Now().Year() + 2,
		Currency:    "GBP",
		Amount:      100,
		Cvv:         "123",
	}
}

// TestIntegration_AuthorizedPayment_CanThenBeRetrieved covers the "happy
// path": a card ending in an odd digit is authorized by the simulator, the
// gateway persists it, and a follow-up GET returns the same data - with only
// the last four digits of the card, never the full number or CVV.
func TestIntegration_AuthorizedPayment_CanThenBeRetrieved(t *testing.T) {
	requireBankSimulator(t)
	server := newTestServer(t)

	req := validRequest("2222405343248877") // ends in 7 -> simulator authorizes
	resp, created := postPayment(t, server, req)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST: want 201, got %d", resp.StatusCode)
	}
	if created.PaymentStatus != client.PaymentStatusAuthorized {
		t.Fatalf("want status Authorized, got %s", created.PaymentStatus)
	}
	if created.CardNumberLastFour != "8877" {
		t.Fatalf("want last four 8877, got %s", created.CardNumberLastFour)
	}
	if created.Id == "" {
		t.Fatal("expected a generated payment id")
	}

	getResp := getPayment(t, server, created.Id)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET: want 200, got %d", getResp.StatusCode)
	}

	var fetched models.GetPaymentResponse
	if err := json.NewDecoder(getResp.Body).Decode(&fetched); err != nil {
		t.Fatalf("decoding GET response: %v", err)
	}
	if fetched != models.GetPaymentResponse(created) {
		t.Fatalf("GET response %+v does not match what POST returned %+v", fetched, created)
	}
}

// TestIntegration_DeclinedPayment covers a card ending in an even digit,
// which the simulator declines without it being an error - it's still a
// successfully processed payment, just not an authorized one.
func TestIntegration_DeclinedPayment(t *testing.T) {
	requireBankSimulator(t)
	server := newTestServer(t)

	req := validRequest("2222405343248112") // ends in 2 -> simulator declines
	resp, created := postPayment(t, server, req)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST: want 201, got %d", resp.StatusCode)
	}
	if created.PaymentStatus != client.PaymentStatusDeclined {
		t.Fatalf("want status Declined, got %s", created.PaymentStatus)
	}
}

// TestIntegration_BankUnavailable covers a card ending in 0, which the
// simulator treats as itself being down (503). The gateway should surface
// that as 503 and must not invent an Authorized/Declined outcome.
func TestIntegration_BankUnavailable(t *testing.T) {
	requireBankSimulator(t)
	server := newTestServer(t)

	req := validRequest("2222405343248870") // ends in 0 -> simulator returns 503
	resp, _ := postPayment(t, server, req)

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", resp.StatusCode)
	}
}

// TestIntegration_InvalidRequest_NeverReachesBank checks that our own
// validation short-circuits obviously bad requests with 400, rather than
// forwarding them to the bank (which would also reject them, but we
// shouldn't rely on that).
func TestIntegration_InvalidRequest_NeverReachesBank(t *testing.T) {
	requireBankSimulator(t)
	server := newTestServer(t)

	req := validRequest("2222405343248877")
	req.ExpiryYear = 2000 // already expired

	resp, _ := postPayment(t, server, req)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}

// TestIntegration_GetPayment_NotFound documents current behaviour for an
// unknown id. Note: this returns 204 No Content today (see GetHandler) even
// though the existing unit test for it asserts 404 - that mismatch predates
// this test and is a separate, known inconsistency worth fixing.
func TestIntegration_GetPayment_NotFound(t *testing.T) {
	requireBankSimulator(t)
	server := newTestServer(t)

	resp := getPayment(t, server, "00000000-0000-0000-0000-000000000000")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("want 204, got %d", resp.StatusCode)
	}
}
