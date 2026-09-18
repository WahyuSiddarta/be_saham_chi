# Basic snapshot residual-income valuation

## Purpose and scope

`POST /api/v1/private/stocks/{ticker}/valuation/residual-income` estimates fair
value per share from current book value and the present value of future residual
income. It is additive to the existing FCF-per-share endpoint and does not save
the valuation.

This endpoint is intentionally a basic snapshot model. It starts from the latest
stored fundamentals and creates a synthetic forecast by linearly fading current
ROE and payout toward terminal assumptions. It is not a historical multi-year
clean-surplus model and does not provide normalized historical ROE, scenarios,
or sensitivity tables.

## Inputs

The service reads these stored values:

- current book value per share and EPS TTM from `perShare`;
- ROE TTM from `managementEffectiveness`, with `profitability` as a compatibility
  fallback;
- dividend events in the trailing twelve months from `dividendHistory`;
- `master_data.indonesia_10_year_bond_yield`, stored as percentage points and used as the IDR risk-free rate; and
- the ticker's exact Yahoo Finance `5y`/`1mo` beta from `stock_betas`, including
  a negative beta.

ROE values with a `%` suffix are always divided by 100, including `0.80%` and
`-0.50%`. A valid empty dividend history, or a valid history with no events in
the trailing twelve months, produces zero dividends and a zero payout ratio.
Malformed dividend events are rejected as unavailable valuation input.

The request supplies decimal `terminal_roe`, `terminal_growth_rate`, and
`equity_risk_premium`. `forecast_years` defaults to 5 and accepts 1 through 10.

## Calculation

For each explicit forecast year:

```text
cost of equity = Indonesia 10-year government bond yield + beta x equity risk premium
earnings per share = forecast ROE x opening book value per share
dividend per share = forecast payout x earnings per share
closing book value = opening book value + earnings - dividends
residual income = (forecast ROE - cost of equity) x opening book value
```

Current ROE and payout fade linearly to terminal ROE and terminal payout. The
terminal retention ratio is `terminal growth / terminal ROE`, and terminal
payout is one minus that retention ratio. Terminal growth must not exceed
terminal ROE and must remain below cost of equity.

Fair value is current book value per share plus the present value of explicit
residual income plus the present terminal value.

## Limitations

Before using the result as an investment recommendation, verify source units,
currency, split-adjusted per-share consistency, special dividends, dilution,
and whether current ROE is representative. A defensible historical model would
also require completed annual history, attributable common equity and earnings,
consistent share counts, dividend reconciliation, clean-surplus adjustments,
normalized ROE, scenarios, and sensitivities.
