# CapitalSight Chi backend

The V2 API is implemented with Chi, shared handlers/services, and one shared
sqlx repository. The Echo project remains available as the contract reference.

## Routing

`cmd/backend/api.go` registers `/api/v1/public` and `/api/v1/private` directly,
with resource groups nested inside them:

- `/public/auth`: registration and login.
- `/public/health`: API health.
- `/public/protected`: token claims example, with its own JWT middleware.
- `/public/docs` and `/public/openapi.yaml`: API documentation.
- `/private`: JWT middleware inherited by every child route.
- `/private/admin`: master data and stock administration.
- `/private/stocks` and `/private/commodities/{commodity}`: market data.
- `/private/portfolios`: portfolio CRUD.
- `/private/portfolio/{portfolio_id}`: cash, bonds, gold, and their transactions.

`cmd/backend/common.go` registers one table per resource, with its path and permission
prefix specified once. Each entry lists its HTTP method, relative path,
permission action, and handler. For example, `portfolio.cash` plus `read` requires
`portfolio.cash.read`. Empty actions use the group's middleware. Special actions,
such as bond valuation requiring `update`, remain explicit.

Shared ACLs sit on their resource groups; different read/create/update/delete
permissions remain on individual operations. Existing V2 URLs and permissions
are retained. The Chi OpenAPI contract is served at `/api/v1/public/openapi.yaml`,
with an interactive viewer at `/api/v1/public/docs`. All routes are under
`/api/v1`; the former root-level health, login, claims, and documentation URLs
are no longer registered. Login uses `/api/v1/public/auth/login`.

## Shared implementation

- `internal/helper.MapSlice` handles list conversion while preserving empty JSON arrays.
- `internal/service/common.go` wraps service errors and discards partial results on failure.
- `internal/repository/common.go` shares struct-row scanning and preserves row errors.

Domain validation, accounting rules, and SQL stay in their feature files.

## Middleware

`internal/middleware` owns request IDs, HTTP request logging, panic recovery,
CORS, bearer-token authentication, and permission checks. The router applies
`RequestID`, `RequestLogger`, `Recover`, and `CORS` in that order, then attaches
`Authenticate`, `RequireRule`, and `CommodityRule` to the relevant route groups.
Request logging and recovery receive the configured logger. JWT signing and
validation remain in `internal/auth`; verified request claims are accessed through
`middleware.ClaimsFromContext` and `middleware.UserIDFromContext`.

## API responses

All JSON API responses use `response.Success` or `response.Fail`:

```json
{"status":"ok","data":{"id":"example"}}
{"status":"ok","data":[]}
{"status":"nok","data":"invalid request body"}
{"status":"nok"}
```

Success always includes `data`; failure omits it when no details are supplied.
List payloads go directly in `data`. Successful deletes return HTTP 200 with
`{"status":"ok","data":null}`. HTTP 204 is reserved for CORS preflight, which
has no body; documentation routes serve HTML/YAML.

This intentionally changes V2's flat object and `error` responses. Before
switching `capitalsight-fe-v2` to Chi, update its consumers (including login,
portfolio details, quotes, and error handling) to read the envelope's `data`.
The frontend and Echo backend have not been changed by this response migration.

## JSON binding

Use `internal/request.BindJSON` with any `io.Reader`; no Echo or Chi context is
required:

```go
var payload CreateStockRequest
if err := request.BindJSON(r.Body, &payload); err != nil {
    // Map the error to an HTTP 400 response in the handler.
}
```

Binding uses Go's JSON v2 semantics, rejects duplicate keys and trailing JSON,
accepts unknown fields, and limits each body to 1 MiB. URL parameters are read
with `chi.URLParam`; query parameters use `r.URL.Query()`.

## Configuration and database

Use `.env.example` for settings. Alongside the existing Chi settings, configure
`GOLD_SYMBOL`, `WTI_OIL_SYMBOL`, `BRENT_OIL_SYMBOL`, and `CORS_ALLOWED_ORIGINS` in
your local `.env`. For existing V2 tokens, retain V2's JWT secret and issuer.

Point `DATABASE_URL` at the current V2 schema or a new development database.
The migration command prepares the required tables and ACL seeds using sqlx. It also adds the
registration `role_id` column to databases created by the earlier Chi prototype.
The migration also owns schema setup for `stealth-scraping`, which shares this
database. It creates `stock_fundamentals`, `stock_financials`, and
`stock_financial_values` with their indexes and existing scraper schema updates.
For legacy fundamentals, it copies section columns into `payload` before removing
those old columns. Financial-history JSON backfill remains in the scraper job;
other historical V2 cleanup is not replayed. An older pre-current-V2 database
still needs its existing migration procedure before use.

Stock fundamentals remain read-only in this backend; the scraper owns writes.
Stock klines retain their database-first Yahoo cache behavior. Registration and
portfolio mutations retain transaction boundaries. Shutdown remains bounded and
closes the HTTP server before the database pool.

## Build and run

Build the two applications separately from the repository root:

```sh
go build -o bin/migrate ./cmd/migrate
go build -o bin/backend ./cmd/backend
```

Run them from the repository root so `.env` and `docs/openapi.yaml` resolve:

```sh
./bin/migrate
# Start the backend only after migration succeeds.
./bin/backend
```

`migrate` initializes tables, applies the existing schema updates, and seeds
roles, ACLs, and master data, then exits. The PostgreSQL database itself must
already exist. It requires only the `DATABASE_*` settings from `.env.example`,
logs to stdout, and exits nonzero on failure. Connection setup has a 10-second
timeout; schema setup has a five-minute timeout. SIGINT/SIGTERM cancels setup
and closes the database pool. Existing setup statements are reused; this does
not introduce versioned migrations or rollback commands.

`backend` only connects to the prepared database and serves the API; startup
no longer changes schema or seeds data. Run `migrate` before the first start
of either the backend or `stealth-scraping`, and whenever deploying schema or seed changes. For development, use
`go run ./cmd/migrate` and `go run ./cmd/backend` respectively. The old
`go run ./cmd` entry point has been replaced.

## Linux AMD64 release build

Build both applications for Ubuntu or another Linux AMD64 host:

```sh
./build-live.sh
```

If Go is not on your PATH, pass its executable explicitly:

```sh
./build-live.sh /Volumes/KyoMac/go/bin/go
```

The script produces `bin/linux-amd64/backend` and `bin/linux-amd64/migrate`
with CGO disabled, baseline AMD64 compatibility, trimmed source paths, and
stripped debug symbols. It also includes `docs/openapi.yaml` and `.env.example`
in that directory. It uses the existing Go cache configuration and may download
missing dependencies or the toolchain required by `go.mod`.

Copy the directory contents to the Ubuntu server, create the production `.env`
there using `.env.example`, and run `./migrate` successfully before starting
`./backend`. Use that directory as the working directory for both applications.
The build script does not run either application or connect to the database.

## Verification

```sh
GOPROXY=off GOSUMDB=off go test ./...
```

Tests use fake providers, local HTTP test servers, and an in-memory SQL driver.
They do not start the application, execute migrations, or contact PostgreSQL or
Yahoo. A live database cutover still needs runtime verification.
