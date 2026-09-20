# Implementation decisions: POST /api/payments

This documents the choices made while implementing payment creation on top
of the original skeleton, and why. The skeleton provided the models'
skeleton, the in-memory repository, the GET endpoint, and the bank
simulator - everything below fills the gap between those and a working
POST.

## API contract

- **Request now carries a full card number.** The original
  `PostPaymentRequest` only had `card_number_last_four`, which a client
  submitting a *new* payment would never actually have (or need) — you
  submit the full card, and the gateway derives the last four for storage
  and the response. `CardNumber` (full) replaces it in the request; the
  response keeps `CardNumberLastFour`.
- **CVV is a string, not a number.** An int would silently drop a leading
  zero (`012`).
- **Card Numebr is a string, not a number.** An int would silently 
    drop a leading zero (`0423`)
- **`PaymentStatus` is a distinct type** (`Authorized` / `Declined` /
  `Rejected`), not a bare string, so invalid values can't be assigned
  without at least a type mismatch warning. 
- **Full card number and CVV are never stored or returned.** They exist
  only for the single call to the bank; the repository and every response
  only ever see the last four digits.

## Validation

Requests are validated *before* anything is sent to the bank
(`internal/models/validation.go`):

| Field | Rule |
|---|---|
| `card_number` | 14-19 numeric characters |
| `expiry_month` / `expiry_year` | valid month, and the combination must be in the future |
| `currency` | one of a small supported-ISO-code allowlist (USD, GBP, EUR) |
| `amount` | positive integer, minor currency unit |
| `cvv` | 3-4 numeric characters |

A request that fails any of these gets `400` with a list of problems, and
the bank is never called. This matters because the bank simulator *also*
rejects incomplete requests (see `imposters/bank_simulator.ejs`), but we
shouldn't rely on a downstream system to catch our own malformed input.

**Left out of scope:** Luhn checksum validation on the card number, a
fuller ISO 4217 currency list, and per-card-scheme length rules. The
allowlist approach was chosen to keep the sketch readable; swap in a
proper currency package if this were going further.

## Talking to the bank (`internal/bank`)

A small dedicated client wraps the HTTP call to the acquiring bank
(the Mountebank imposter in this challenge):

- **Base URL is configurable** via the `BANK_SIMULATOR_URL` environment
  variable, defaulting to `http://localhost:8080` (matching
  `docker-compose.yml`). This is what makes the client swappable for tests.
- **5 second timeout** on the HTTP client, so a hung bank doesn't hang the
  gateway indefinitely.
- **"Unreachable" is a distinct outcome from "declined".** If the bank
  can't be reached, times out, or returns something we don't understand,
  `Authorize` returns an `error` rather than a false `Declined`. The
  handler maps that to `503` and **stores nothing** — we don't actually
  know what happened to the payment, so we shouldn't invent an outcome for
  it.

## Handler flow (`internal/handlers/payments_handler.go`)

```
decode JSON → validate → call bank → map to status → persist → respond
     │            │           │
   400 on      400 on      503 on
  bad JSON   validation    bank error
             failure      (nothing stored)
```

- **`201 Created`** on a successfully *processed* payment — whether that
  payment was Authorized or Declined. Both are legitimate outcomes of a
  request that was handled correctly; only validation failures and bank
  unavailability are treated as request-level errors (`400` / `503`).
- **Errors are returned as `{"errors": ["...", "..."]}`** — a flat list of
  human-readable strings, rather than per-field structured errors, again
  favouring simplicity over a fuller error schema.


## Testability: `Api.Handler()`

`Api.router` was and remains unexported, but integration tests need to
drive the *real* router. Rather than exposing the field or forcing tests
through `Run`'s OS-port-binding lifecycle, `Api` gained one exported
method:

```go
func (a *Api) Handler() http.Handler { return a.router }
```

This lets a test do `httptest.NewServer(a.Handler())` and get the actual
production routing, middleware, and handler wiring, without managing a
real port or process lifecycle.

## Testing strategy: unit vs. integration

Two layers, deliberately kept separate:

- **Unit tests** (`internal/handlers/payments_post_handler_test.go`) fake
  the bank with an `httptest.Server` under the test's control. They're
  fast, deterministic, cover edge cases precisely (authorized, declined,
  validation-rejected, bank-unreachable), and run as part of the default
  `go test ./...`.
- **Integration tests** (`internal/api/api_integration_test.go`) build the
  real `Api`, with its real bank client, and talk to the *actual* bank
  simulator over HTTP. They're gated behind a build tag so they don't run
  by default and don't break CI when the simulator isn't up:

  ```
  docker-compose up -d
  go test -tags=integration ./internal/api/...
  ```

  If the simulator isn't reachable on `localhost:8080`, each integration
  test calls `t.Skip` with an explanatory message rather than failing with
  a confusing connection error.

  These tests cover the full lifecycle: POST an authorized payment, then
  GET it back and confirm the two responses match; POST a declined
  payment; POST to a "down" card and confirm `503` with nothing stored;
  POST an invalid request and confirm it's rejected before the bank is
  ever called; GET an unknown id.

## Known gaps / explicitly out of scope

- **The GET endpoint's not-found status is `204`, not `404`.** The
  existing unit test for GET actually asserts `404`, which doesn't match
  the handler's real behaviour — that mismatch predates this work. The new
  integration test documents and asserts the *actual* current behaviour
  (`204`) rather than silently changing it; fixing the inconsistency
  itself is a separate, one-line change if wanted.
- **No idempotency key / duplicate-submission handling.** A retried POST
  creates a second payment.
- **No authentication/authorization** on either endpoint.
- **In-memory storage only** — payments don't survive a restart. This was
  already true of the provided skeleton and wasn't in scope to change.
- **Swagger annotations weren't added** for the new POST endpoint/fields;
  the generated docs still only describe `/ping`.