package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aradia23/checkout-payment-gateway-challengeo/docs"
	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/handlers"
	httpSwagger "github.com/swaggo/http-swagger"
)

type pong struct {
	Message string `json:"message"`
}

// PingHandler returns an http.HandlerFunc that handles HTTP Ping GET requests.
func (a *Api) PingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(pong{Message: "pong"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// SwaggerHandler returns an http.HandlerFunc that handles HTTP Swagger related requests.
func (a *Api) SwaggerHandler() http.HandlerFunc {
	return httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("http://%s/swagger/doc.json", docs.SwaggerInfo.Host)),
	)
}

// GetPaymentHandler returns an http.HandlerFunc that handles Payments GET requests.
func (a *Api) GetPaymentHandler() http.HandlerFunc {
	h := handlers.NewPaymentsHandler(a.paymentsRepo, a.bankClient)

	return h.GetHandler()
}

// GetPaymentHandler returns an http.HandlerFunc that handles Payments GET requests.
func (a *Api) PostPaymentHandler() http.HandlerFunc {
	h := handlers.NewPaymentsHandler(a.paymentsRepo, a.bankClient)

	return h.PostHandler()
}
