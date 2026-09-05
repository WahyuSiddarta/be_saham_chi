# Rewrite Rules: `be_saham_v2` to `be_saham_chi`

Use these rules whenever code or behavior is rewritten from `be_saham_v2` into
`be_saham_chi`.

## 1. Follow `be_saham_chi/AGENTS.md`

- Read and follow the nearest applicable `AGENTS.md` before making changes.
- Treat `be_saham_chi` as the source of truth for architecture, naming,
  initialization, environment variables, application lifecycle, and validation.
- Do not copy the structure of `be_saham_v2` when it conflicts with these rules.
- Preserve existing API and database behavior unless the rewrite task explicitly
  requires a contract change.
- Keep each rewrite focused. Do not migrate unrelated code in the same change.

In particular:

- Create handlers through the shared `handler.New(...)` initializer.
- Create repositories through the shared `repository.New(...)` initializer.
- Domain services may have separate constructors when their dependencies differ.
- Any server or worker must support the graceful-shutdown lifecycle required by
  `AGENTS.md`.
- Environment-specific values belong in `.env`, with safe examples added to
  `.env.example`. Never hardcode or commit secrets.

## 2. Use `sqlx` for repository code

- Repository structs must use the shared `*sqlx.DB` dependency.
- Use the context-aware `sqlx` methods appropriate to the query, such as
  `GetContext`, `SelectContext`, `ExecContext`, and `NamedExecContext`.
- Use `sqlx.Tx` for transactions and pass `context.Context` through repository
  calls.
- Map query results into typed structs with explicit `db` tags.
- Keep SQL inside the repository layer. Handlers and services must not query the
  database directly.
- Do not introduce another database abstraction, ORM, or separate connection
  pool unless the task explicitly requires it.
- Using standard-library database types such as `sql.ErrNoRows` or `sql.Null*`
  is acceptable; query execution must still go through `sqlx`.

## 3. Place functions by layer and domain

Before moving a function, identify its responsibility and put it in the matching
package and domain file. Do not preserve the scattered placement from
`be_saham_v2`.

| Responsibility | Package | File example |
| --- | --- | --- |
| HTTP parsing, validation, and response mapping | `internal/handler` | `portfolio.go` |
| Business rules and workflow orchestration | `internal/service` | `portfolio.go` |
| SQL and persistence mapping | `internal/repository` | `portfolio.go` |
| JWT signing and validation | `internal/auth` | `jwt.go` |
| HTTP middleware, authentication, authorization, and request context | `internal/middleware` | `auth.go`, `common.go`, `cors.go`, `recover.go`, `logger.go` |
| Shared JSON response writing | `internal/response` | `json.go` |
| Application wiring and route registration | `cmd` | `api.go` |

File-placement rules:

- Name domain files after the feature they own, for example `auth.go`,
  `portfolio.go`, or `stock.go`.
- Keep shared structs and shared initializers in `common.go`; do not turn
  `common.go` into a catch-all for domain behavior.
- Keep an unexported helper beside the exported function that owns it when the
  helper is used only in that file.
- Move a genuinely cross-domain helper into the narrowest appropriate shared
  package only after more than one domain needs it.
- Keep related request/response types, errors, interfaces, and functions close
  to the domain code that uses them.
- Within a file, prefer this order: package-level constants and errors, types,
  constructors, exported functions or methods, then unexported helpers.
- If a domain file becomes difficult to navigate, split it by a clear
  responsibility such as `portfolio_transactions.go` and
  `portfolio_holdings.go`; do not create arbitrary files or one file per small
  function.

## 4. API response contract

- Use `response.Success` and `response.Fail` for every JSON API response,
  including middleware and router errors.
- Success has `status: "ok"` and the payload inside `data`.
- Failure has `status: "nok"`; optional details go inside `data`.
- Domain response types contain payload fields only. Do not add another
  status/data envelope or leave payload fields at the top level.
- Successful deletes return HTTP 200 with `data: null`, so they carry the same
  envelope. CORS preflight and non-JSON documentation are protocol exceptions.


## Rewrite checklist

Before considering a rewritten feature complete, confirm that:

- the applicable `AGENTS.md` instructions are satisfied;
- repository access uses the shared `sqlx` repository;
- each function is in the correct layer and domain file;
- no duplicated handler or repository constructor was introduced;
- API and database contracts remain compatible unless a change was requested;
- changed Go files are formatted, and safe static checks have been run without
  starting the backend or requiring a live database.
