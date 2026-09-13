# Residual Income Model Implementation Review

Review date: 2026-09-10

Status: backend snapshot-model corrections implemented; live verification and
the separate frontend integration remain outstanding.

## Resolution

The backend follow-up keeps this endpoint as an explicitly named basic snapshot
RIM and addresses the production defects within that scope:

- ROE normalization now follows the source `%` suffix, including values at or
  below 1% and negative percentages;
- a valid empty dividend history or no TTM events produces zero dividends,
  while malformed events remain invalid input;
- missing fundamentals, BI rate, and beta remain client-facing unavailable
  inputs, while unexpected repository failures retain their cause and map to
  HTTP 500;
- focused service and handler tests cover the reviewed boundary, default,
  calculation, repository-error, and response-envelope cases; and
- `docs/openapi.yaml` and `docs/residual-income-valuation.md` now define the API
  contract and clearly state the snapshot model's methodology and limitations.

The larger historical clean-surplus model and the separate frontend RIM panel
are distinct follow-up work, not implicit behavior of this endpoint.

## Summary

The current implementation provides a coherent basic, snapshot-based Residual
Income Model (RIM). Its central calculation uses opening book value per share,
rolls book value forward through retained earnings, preserves the stored beta,
discounts residual income using cost of equity, and applies a terminal value
whose retention ratio is consistent with terminal growth and terminal ROE.

It is not yet the planned defensible multi-year implementation. It also contains
two data-handling defects that can materially affect results or reject otherwise
valid stocks.

## Findings

### High: ROE percentages below or equal to 1% are interpreted incorrectly

`readResidualIncomeMetrics` parses a percentage string as a plain number and
divides it by 100 only when its absolute value is greater than 1.

For example:

- `21.47%` becomes `0.2147`, which is correct.
- `0.80%` remains `0.80`, which is interpreted as 80% instead of 0.80%.
- `-0.50%` remains `-0.50`, which is interpreted as -50% instead of -0.50%.

Percentage conversion must use the source unit or `%` suffix rather than the
numeric magnitude.

Relevant code: `internal/service/stock_residual_income.go`, ROE parsing in
`readResidualIncomeMetrics`.

### High: zero-dividend companies are rejected

The implementation requires at least one parseable dividend event during the
trailing twelve months. A company with no dividend events receives an invalid
assumptions error even though a zero payout ratio is a valid RIM input.

An absent TTM dividend event should be distinguished from malformed dividend
data. A valid empty history should produce dividend per share of zero and a
zero current payout ratio.

Relevant code: `internal/service/stock_residual_income.go`, dividend-history
handling in `readResidualIncomeMetrics` and the shared `dividendPerShareTTM`
helper.

### High: the implementation is a basic snapshot RIM, not the target multi-year RIM

The service currently reads only the latest:

- book value per share;
- EPS TTM;
- ROE TTM;
- dividend events from the trailing twelve months;
- BI rate; and
- stored stock beta.

It then creates a synthetic forecast by linearly fading current ROE and payout
to terminal assumptions. It does not use completed annual history, attributable
common equity and earnings, consistent share counts, dividend reconciliation,
clean-surplus adjustments, normalized ROE, scenarios, or sensitivities.

This is acceptable only if the endpoint and documentation explicitly describe
it as a basic snapshot model. It should not be presented as the defensible
multi-year RIM originally planned.

### Medium: repository failures can be returned as HTTP 400

Failures from `GetFundamentals` and `GetMasterData` are converted into
`ErrInvalidResidualIncomeAssumptions`. The handler consequently returns HTTP
400 even when the real cause is a database or repository failure.

Missing valuation inputs may return a client-facing unavailable/validation
response. Unexpected repository failures should retain their underlying error
and produce HTTP 500.

Relevant code: `internal/service/stock_residual_income.go`, repository reads in
`CalculateResidualIncome`; `internal/handler/stock_valuation.go`, error mapping
in `CalculateStockResidualIncome`.

### Medium: API and frontend integration are incomplete

The backend route is registered at:

`POST /api/v1/private/stocks/{ticker}/valuation/residual-income`

However:

- `docs/openapi.yaml` does not define the endpoint, request, or response schema;
- there is no dedicated residual-income methodology/API document; and
- `capitalsight-fe-v2` has only the FCF valuation hook and panel, not a separate
  RIM hook and panel.

The RIM method must remain additive and separate from the existing FCF method.

### Medium: automated coverage is insufficient

Current tests cover one synthetic successful calculation and one inconsistent
terminal-assumption rejection. Additional focused cases should cover:

- an actual representative stored fundamentals payload;
- ROE values such as `0.80%` and `-0.50%`;
- an empty but valid dividend history;
- malformed dividend history;
- repository/database errors;
- missing beta and missing fundamentals;
- default `forecast_years` behavior;
- payout ratios above 100% and their book-value effect;
- exact forecast and terminal-value results from an independently calculated
  example; and
- handler response status and envelope behavior.

## Correct parts

The following aspects of the current implementation are internally coherent:

- Existing FCF behavior and route remain separate.
- Stored beta is preserved exactly, including negative beta.
- Cost of equity is calculated as risk-free rate plus beta multiplied by equity
  risk premium.
- Residual income uses opening book value per share.
- Book value rolls forward as opening book value plus earnings minus dividends.
- Terminal retention is calculated as terminal growth divided by terminal ROE.
- Terminal payout is one minus terminal retention.
- The terminal residual income is based on the next opening book value and is
  discounted from the end of the explicit forecast period.
- Fair value adds current book value, present explicit residual income, and the
  present terminal value.
- Validation requires terminal growth to remain below cost of equity.

## Validation performed during review

- `gofmt -d` produced no formatting differences for the new RIM service, tests,
  and handler changes.
- `git diff --check` passed.
- Stored metric names were compared with the checked-in representative
  `stealth-scraping/debug.json` payload.

Go tests, backend startup, live database access, migration execution, and live
endpoint verification were not performed because they require explicit approval
under the workspace instructions.

## Recommended completion order

1. Correct percentage parsing and add boundary tests.
2. Treat a valid empty dividend history as zero payout.
3. Separate missing-input errors from repository/infrastructure errors.
4. Decide whether this endpoint is explicitly a basic snapshot RIM or extend it
   to the planned historical multi-year model.
5. Add the OpenAPI contract and methodology documentation.
6. Add the separate frontend RIM integration.
7. Run focused Go tests and verify an independently calculated example.
8. With explicit approval, validate a representative live payload and endpoint.
