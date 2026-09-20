package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/client"
	"github.com/aradia23/checkout-payment-gateway-challengeo/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/sync/errgroup"
)

type Api struct {
	router       *chi.Mux
	paymentsRepo *repository.PaymentsRepository
	bankClient   *client.Client
}

func New() *Api {
	a := &Api{}
	a.paymentsRepo = repository.NewPaymentsRepository()
	a.bankClient = client.NewClient(bankBaseUrl())
	a.setupRouter()

	return a
}

func (a *Api) Run(ctx context.Context, addr string) error {
	httpServer := &http.Server{
		Addr:        addr,
		Handler:     a.router,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		<-ctx.Done()
		fmt.Printf("shutting down HTTP server\n")
		return httpServer.Shutdown(ctx)
	})

	g.Go(func() error {
		fmt.Printf("starting HTTP server on %s\n", addr)
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	return g.Wait()
}

func (a *Api) setupRouter() {
	a.router = chi.NewRouter()
	a.router.Use(middleware.Logger)

	a.router.Get("/ping", a.PingHandler())
	a.router.Get("/swagger/*", a.SwaggerHandler())

	a.router.Get("/api/payments/{id}", a.GetPaymentHandler())

	a.router.Post("/api/payments/", a.PostPaymentHandler())
}

func bankBaseUrl() string {
	if env := os.Getenv("BANK_BASE_URL"); env != "" {
		return env
	}
	return "http://localhost:8080"
}
