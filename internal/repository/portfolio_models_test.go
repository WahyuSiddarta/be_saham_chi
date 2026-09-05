package repository

import (
	"testing"
	"time"
)

func TestPortfolioCashTransactionFlowAmounts(t *testing.T) {
	tests := []struct {
		name          string
		tx            PortfolioCashTransaction
		wantCashFlow  float64
		wantCostFlow  float64
		wantPnLEffect float64
	}{
		{
			name: "deposit adds cash and cost with neutral pnl",
			tx: PortfolioCashTransaction{
				TransactionType: CashTransactionDeposit,
				Amount:          1000,
				CostAmount:      1000,
			},
			wantCashFlow:  1000,
			wantCostFlow:  1000,
			wantPnLEffect: 0,
		},
		{
			name: "withdrawal removes cash and cost with neutral pnl",
			tx: PortfolioCashTransaction{
				TransactionType: CashTransactionWithdrawal,
				Amount:          1000,
				CostAmount:      1000,
			},
			wantCashFlow:  -1000,
			wantCostFlow:  -1000,
			wantPnLEffect: 0,
		},
		{
			name: "interest adds pnl",
			tx: PortfolioCashTransaction{
				TransactionType: CashTransactionInterest,
				Amount:          100,
			},
			wantCashFlow:  100,
			wantCostFlow:  0,
			wantPnLEffect: 100,
		},
		{
			name: "tax lowers pnl",
			tx: PortfolioCashTransaction{
				TransactionType: CashTransactionTax,
				Amount:          25,
			},
			wantCashFlow:  -25,
			wantCostFlow:  0,
			wantPnLEffect: -25,
		},
		{
			name: "maturity records difference between received cash and cost",
			tx: PortfolioCashTransaction{
				TransactionType: CashTransactionMaturity,
				Amount:          1100,
				CostAmount:      1000,
			},
			wantCashFlow:  1100,
			wantCostFlow:  1000,
			wantPnLEffect: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tx.CashFlowAmount(); got != tt.wantCashFlow {
				t.Fatalf("CashFlowAmount() = %v, want %v", got, tt.wantCashFlow)
			}
			if got := tt.tx.CostFlowAmount(); got != tt.wantCostFlow {
				t.Fatalf("CostFlowAmount() = %v, want %v", got, tt.wantCostFlow)
			}
			if got := tt.tx.PnLEffectAmount(); got != tt.wantPnLEffect {
				t.Fatalf("PnLEffectAmount() = %v, want %v", got, tt.wantPnLEffect)
			}
		})
	}
}

func TestPortfolioCashAccountRecalculatePnL(t *testing.T) {
	account := PortfolioCashAccount{
		Quantity:  1125,
		TotalCost: 1000,
	}

	account.RecalculatePnL()

	if account.UnrealizedPnL != 125 {
		t.Fatalf("UnrealizedPnL = %v, want 125", account.UnrealizedPnL)
	}
	if account.TotalPnL != 125 {
		t.Fatalf("TotalPnL = %v, want 125", account.TotalPnL)
	}
	if account.TotalPnLPercent != 12.5 {
		t.Fatalf("TotalPnLPercent = %v, want 12.5", account.TotalPnLPercent)
	}
}

func TestPortfolioCashRecalculatePnL(t *testing.T) {
	cash := PortfolioCash{
		TotalCash: 22225300,
		TotalCost: 22200000,
	}

	cash.RecalculatePnL()

	if cash.UnrealizedPnL != 25300 {
		t.Fatalf("UnrealizedPnL = %v, want 25300", cash.UnrealizedPnL)
	}
	if cash.TotalPnL != 25300 {
		t.Fatalf("TotalPnL = %v, want 25300", cash.TotalPnL)
	}
	wantPercent := (25300.0 / 22200000.0) * 100
	if cash.TotalPnLPercent != wantPercent {
		t.Fatalf("TotalPnLPercent = %v, want %v", cash.TotalPnLPercent, wantPercent)
	}
}

func TestAdjustmentIsNotCashTransactionType(t *testing.T) {
	if IsCashTransactionType("adjustment") {
		t.Fatal("adjustment must not be accepted as a cash transaction type")
	}
}

func TestPortfolioBondAccountRecalculatePnL(t *testing.T) {
	account := PortfolioBondAccount{
		PrincipalAmount: 10000000,
		TotalCost:       10000000,
		MarketValue:     9800000,
	}

	account.RecalculatePnL()

	if account.UnrealizedPnL != -200000 {
		t.Fatalf("UnrealizedPnL = %v, want -200000", account.UnrealizedPnL)
	}
	if account.TotalPnL != -200000 {
		t.Fatalf("TotalPnL = %v, want -200000", account.TotalPnL)
	}
	if account.TotalPnLPercent != -2 {
		t.Fatalf("TotalPnLPercent = %v, want -2", account.TotalPnLPercent)
	}
}

func TestPortfolioBondRecalculatePnL(t *testing.T) {
	bond := PortfolioBond{
		PrincipalAmount: 15000000,
		TotalCost:       15100000,
		MarketValue:     15200000,
	}

	bond.RecalculatePnL()

	if bond.UnrealizedPnL != 100000 {
		t.Fatalf("UnrealizedPnL = %v, want 100000", bond.UnrealizedPnL)
	}
	if bond.TotalPnL != 100000 {
		t.Fatalf("TotalPnL = %v, want 100000", bond.TotalPnL)
	}
	wantPercent := (100000.0 / 15100000.0) * 100
	if bond.TotalPnLPercent != wantPercent {
		t.Fatalf("TotalPnLPercent = %v, want %v", bond.TotalPnLPercent, wantPercent)
	}
}

func TestBondTransactionTypes(t *testing.T) {
	validTypes := []string{
		BondTransactionBuy,
		BondTransactionSell,
		BondTransactionCoupon,
		BondTransactionFee,
		BondTransactionTax,
		BondTransactionMaturity,
	}
	for _, transactionType := range validTypes {
		if !IsBondTransactionType(transactionType) {
			t.Fatalf("%s should be accepted as a bond transaction type", transactionType)
		}
	}
	if IsBondTransactionType(CashTransactionDeposit) {
		t.Fatal("deposit must not be accepted as a bond transaction type")
	}
}

func TestBondCouponScheduleSummarySemiannual(t *testing.T) {
	term := PortfolioBondTerm{
		IssueDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		MaturityDate:    time.Date(2029, 1, 15, 0, 0, 0, 0, time.UTC),
		AnnualRate:      5,
		CouponFrequency: "semiannual",
	}

	summary := term.CouponScheduleSummary(10000000, time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC))

	if summary.CouponPaymentsPerYear != 2 {
		t.Fatalf("CouponPaymentsPerYear = %v, want 2", summary.CouponPaymentsPerYear)
	}
	if summary.CouponAmountPerPeriod != 250000 {
		t.Fatalf("CouponAmountPerPeriod = %v, want 250000", summary.CouponAmountPerPeriod)
	}
	wantDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	if !summary.NextCouponDate.Equal(wantDate) {
		t.Fatalf("NextCouponDate = %v, want %v", summary.NextCouponDate, wantDate)
	}
	if summary.IsNextCouponAtMaturity {
		t.Fatal("IsNextCouponAtMaturity = true, want false")
	}
	if summary.PrincipalReturnedAmount != 0 {
		t.Fatalf("PrincipalReturnedAmount = %v, want 0", summary.PrincipalReturnedAmount)
	}
}

func TestBondCouponScheduleSummaryMaturity(t *testing.T) {
	term := PortfolioBondTerm{
		IssueDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		MaturityDate:    time.Date(2029, 1, 15, 0, 0, 0, 0, time.UTC),
		AnnualRate:      5,
		CouponFrequency: "semiannual",
	}

	summary := term.CouponScheduleSummary(10000000, time.Date(2028, 7, 15, 0, 0, 0, 0, time.UTC))

	wantDate := time.Date(2029, 1, 15, 0, 0, 0, 0, time.UTC)
	if !summary.NextCouponDate.Equal(wantDate) {
		t.Fatalf("NextCouponDate = %v, want %v", summary.NextCouponDate, wantDate)
	}
	if !summary.IsNextCouponAtMaturity {
		t.Fatal("IsNextCouponAtMaturity = false, want true")
	}
	if summary.PrincipalReturnedAmount != 10000000 {
		t.Fatalf("PrincipalReturnedAmount = %v, want 10000000", summary.PrincipalReturnedAmount)
	}
}

func TestBondCouponScheduleSummaryMatured(t *testing.T) {
	term := PortfolioBondTerm{
		IssueDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		MaturityDate:    time.Date(2029, 1, 15, 0, 0, 0, 0, time.UTC),
		AnnualRate:      5,
		CouponFrequency: "semiannual",
	}

	summary := term.CouponScheduleSummary(10000000, time.Date(2029, 1, 15, 0, 0, 0, 0, time.UTC))

	if !summary.NextCouponDate.IsZero() {
		t.Fatalf("NextCouponDate = %v, want zero", summary.NextCouponDate)
	}
	if summary.CouponAmountPerPeriod != 250000 {
		t.Fatalf("CouponAmountPerPeriod = %v, want 250000", summary.CouponAmountPerPeriod)
	}
}

func TestBondCouponFrequency(t *testing.T) {
	if !IsBondCouponFrequency("semiannual") {
		t.Fatal("semiannual should be accepted")
	}
	if IsBondCouponFrequency("weekly") {
		t.Fatal("weekly should be rejected")
	}
}
