# CapitalSight Chi backend

The V2 API is implemented with Chi, shared handlers/services, and one shared
sqlx repository. The Echo project remains available as the contract reference.

## Routing

`cmd/api.go` registers nested `Route` groups under `/api/v1`:

- `/public/auth`: registration and login.
- `/private`: JWT middleware inherited by every child route.
- `/private/admin`: master data and stock administration.
- `/private/stocks` and `/private/commodities/{commodity}`: market data.
- `/private/portfolios`: portfolio CRUD.
- `/private/portfolio/{portfolio_id}`: cash, bonds, gold, and their transactions.

Shared ACLs sit on their resource groups; different read/create/update/delete
permissions remain on individual operations. Existing V2 URLs and permissions
are retained. The Chi OpenAPI contract is served at `/openapi.yaml`, with an
interactive viewer at `/docs`.

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
Startup prepares the required tables and ACL seeds using sqlx. It also adds the
registration `role_id` column to databases created by the earlier Chi prototype.
Historical V2 table/column deletion and data backfills are not replayed: an older
pre-current-V2 database needs its existing migration procedure before use.

Stock fundamentals remain read-only in this backend; the scraper owns writes.
Stock klines retain their database-first Yahoo cache behavior. Registration and
portfolio mutations retain transaction boundaries. Shutdown remains bounded and
closes the HTTP server before the database pool.

## Verification

```sh
GOPROXY=off GOSUMDB=off go test ./...
```

Tests use fake providers, local HTTP test servers, and an in-memory SQL driver.
They do not start the application, execute migrations, or contact PostgreSQL or
Yahoo. A live database cutover still needs runtime verification.
