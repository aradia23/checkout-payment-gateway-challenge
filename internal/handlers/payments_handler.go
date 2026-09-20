package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/client"
	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/models"
	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/repository"

	"github.com/go-chi/chi/v5"
)

type PaymentsHandler struct {
	storage    *repository.PaymentsRepository
	bankClient *client.Client
}

func NewPaymentsHandler(storage *repository.PaymentsRepository, bankClient *client.Client) *PaymentsHandler {
	return &PaymentsHandler{
		storage:    storage,
		bankClient: bankClient,
	}
}

// GetHandler returns an http.HandlerFunc that handles HTTP GET requests.
// It retrieves a payment record by its ID from the storage.
// The ID is expected to be part of the URL.
func (h *PaymentsHandler) GetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := uuid.Parse(id); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		payment := h.storage.GetPayment(id)

		if payment != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(payment); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func (ph *PaymentsHandler) PostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.PostPaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid request payload",
			})
			return
		}

		validationErrors := req.Validate()
		if len(validationErrors) > 0 {
			writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
				"errors": validationErrors,
			})
			return
		}

		// Create a bank authorization request
		bankReq := createBankAuthorizationRequest(&req)
		bankResp, err := ph.bankClient.Authorize(bankReq)
		if err != nil {
			writeJSONResponse(w, http.StatusServiceUnavailable, map[string]interface{}{
				"error": "Failed to authorize payment",
			})
			return
		}

		status := client.PaymentStatusDeclined
		if bankResp.Authorized {
			status = client.PaymentStatusAuthorized
		}

		payment, err := ph.storePayment(&req, status)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to store payment",
			})
			return
		}

		writeJSONResponse(w, http.StatusCreated, payment)
	}
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func createBankAuthorizationRequest(pr *models.PostPaymentRequest) client.AutherizaionRequest {
	return client.AutherizaionRequest{
		CardNumber: pr.CardNumber,
		ExpiryDate: fmt.Sprintf("%02d/%d", pr.ExpiryMonth, pr.ExpiryYear),
		Currency:   pr.Currency,
		Amount:     pr.Amount,
		Cvv:        pr.Cvv,
	}
}

func (ph *PaymentsHandler) storePayment(req *models.PostPaymentRequest, status string) (*models.PostPaymentResponse, error) {
	payment := &models.PostPaymentResponse{
		Id:                 uuid.New().String(),
		PaymentStatus:      status,
		CardNumberLastFour: getLastFourDigits(req.CardNumber),
		ExpiryMonth:        req.ExpiryMonth,
		ExpiryYear:         req.ExpiryYear,
		Currency:           req.Currency,
		Amount:             req.Amount,
	}

	ph.storage.AddPayment(*payment)
	return payment, nil
}

func getLastFourDigits(cardNumber string) string {
	return cardNumber[len(cardNumber)-4:]

}
