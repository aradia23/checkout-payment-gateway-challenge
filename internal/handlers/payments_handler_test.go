package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	bank "github.com/aradia23/checkout-payment-gateway-challengeo/internal/client"
	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/models"
	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// fakeBank spins up a real HTTP server standing in for the acquiring bank
// simulator, so PostHandler is exercised end-to-end through the bank client.
func fakeBank(t *testing.T, status int, body string) *bank.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return bank.NewClient(server.URL)
}

func doPost(handler http.HandlerFunc, req models.PostPaymentRequest) *httptest.ResponseRecorder {
	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler(w, httpReq)
	return w
}

func validRequest() models.PostPaymentRequest {
	return models.PostPaymentRequest{
		CardNumber:  "2222405343248877",
		ExpiryMonth: 4,
		ExpiryYear:  2030,
		Currency:    "GBP",
		Amount:      100,
		Cvv:         "123",
	}
}

func TestPostPaymentHandler_Authorized(t *testing.T) {
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fakeBank(t, http.StatusOK, `{"authorized":true,"authorization_code":"abc-123"}`))

	w := doPost(h.PostHandler(), validRequest())

	var resp models.PostPaymentResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusCreated, w.Code)

	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, bank.PaymentStatusAuthorized, resp.PaymentStatus)
	assert.Equal(t, "8877", resp.CardNumberLastFour)
	assert.NotEmpty(t, resp.Id)
	assert.NotNil(t, repo.GetPayment(resp.Id))
}

func TestPostPaymentHandler_Declined(t *testing.T) {
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fakeBank(t, http.StatusOK, `{"authorized":false,"authorization_code":""}`))

	w := doPost(h.PostHandler(), validRequest())

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.PostPaymentResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, bank.PaymentStatusDeclined, resp.PaymentStatus)
}

func TestPostPaymentHandler_ValidationFailsBeforeCallingBank(t *testing.T) {
	repo := repository.NewPaymentsRepository()
	bankCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bankCalled = true
	}))
	defer server.Close()

	h := NewPaymentsHandler(repo, bank.NewClient(server.URL))

	invalid := validRequest()
	invalid.ExpiryYear = 2000 // expired

	w := doPost(h.PostHandler(), invalid)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, bankCalled, "the bank should never be called for a request that fails validation")
}

func TestPostPaymentHandler_BankUnavailable(t *testing.T) {
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fakeBank(t, http.StatusServiceUnavailable, ""))

	w := doPost(h.PostHandler(), validRequest())

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetPaymentHandler(t *testing.T) {
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fakeBank(t, http.StatusOK, `{"authorized":true,"authorization_code":"abc-123"}`))

	// First, create a payment to retrieve
	w := doPost(h.PostHandler(), validRequest())
	assert.Equal(t, http.StatusCreated, w.Code)

	var createdResp models.PostPaymentResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &createdResp))

	// Now, retrieve the payment using the GetHandler
	req := httptest.NewRequest(http.MethodGet, "/api/payments/"+createdResp.Id, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", createdResp.Id)

	req = req.WithContext(context.WithValue(
		req.Context(),
		chi.RouteCtxKey,
		rctx,
	))
	w = httptest.NewRecorder()
	h.GetHandler()(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var getResp models.GetPaymentResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &getResp))
	assert.Equal(t, createdResp.Id, getResp.Id)
	assert.Equal(t, createdResp.PaymentStatus, getResp.PaymentStatus)
	assert.Equal(t, createdResp.CardNumberLastFour, getResp.CardNumberLastFour)
	assert.Equal(t, createdResp.ExpiryMonth, getResp.ExpiryMonth)
	assert.Equal(t, createdResp.ExpiryYear, getResp.ExpiryYear)
	assert.Equal(t, createdResp.Currency, getResp.Currency)
	assert.Equal(t, createdResp.Amount, getResp.Amount)
}
