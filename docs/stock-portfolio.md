# Stock portfolio

The stock ledger shares `assets`, `portfolio_accounts`, `portfolio_transactions`, and `portfolio_holdings` with existing asset classes. No new tables are required. `EnsureAuthTables`, already called by `EnsureTables` in the migration application, seeds `portfolio.stock.read/create/update/delete` using the existing role assignment policy. Run the migration application through the normal approved deployment process, then log in again to refresh frontend permissions. Never initialize this data during API startup.

Routes under `/api/v1/private/portfolio/{portfolio_id}/stocks`:

- `GET /`: holdings per account/ticker, including closed positions and realized PnL.
- `GET /transactions`: complete transaction ledger, newest first.
- `POST /transactions`: create a buy or sell.
- `PUT /transactions/{transaction_id}`: replace transaction input, preserving its time if omitted.
- `DELETE /transactions/{transaction_id}`: remove and replay.

See `openapi.yaml` for request and response schemas. Supply an account ID or account name and a ticker from the stock master. Account names reuse an exact name within the portfolio or create a broker account. Inactive master tickers remain usable so historical holdings can be recorded and closed. Quantities are whole shares; prices and all monetary fields are IDR. The UI shows 100 shares as one lot but does not require multiples of 100.

Buy cost is gross value plus fees and taxes. Sales remove weighted average cost and recognize net proceeds minus removed cost as realized PnL. All writes lock the owned portfolio row, replay by transaction date / creation time / transaction ID, and commit the transaction and rebuilt holdings together. Edits that change account or ticker rebuild both ledgers. A negative historical balance rolls the entire operation back, including deletion of an earlier buy. The replay engine is shared with gold; the stock adapter maps insufficient-quantity errors to the stock domain.

Each distinct open ticker is quoted once per holdings request using the existing Yahoo provider, with at most four concurrent calls and an eight-second request budget. Failed prices and their derived values remain null. Closed positions have zero market value and retain realized PnL. `price_fetched_at` is retrieval time, not trade time. The frontend does not sum partial market quotes into an apparently complete stock valuation. Its query remains fresh for one minute and offers manual refresh.

This version records buy/sell trades; dividends, corporate actions, automated cash settlement, historical stock snapshots, and broker imports are outside this flow. Existing cash and bond snapshot charts remain unchanged. The ledger list is currently unpaginated, consistent with the other portfolio ledgers.

Offline coverage includes input validation, quote deduplication and failure behavior, owner/asset boundaries, weighted-average cost, and historical oversell rejection. Live PostgreSQL locking, SQL execution, migration, Yahoo availability, and authenticated browser flows require separate environment validation.
