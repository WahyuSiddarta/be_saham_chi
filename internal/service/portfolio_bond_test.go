package service

import (
	"errors"
	"testing"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

func TestValidateBondAssetInputAcceptsSemiannualCouponFrequency(t *testing.T) {
	input := BondAssetInput{
		Symbol:          "ORI025",
		Name:            "ORI025",
		IssueDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		MaturityDate:    time.Date(2029, 1, 15, 0, 0, 0, 0, time.UTC),
		AnnualRate:      5,
		CouponFrequency: "semiannual",
		PrincipalValue:  10000000,
	}

	repoInput, err := validateBondAssetInput(input)
	if err != nil {
		t.Fatalf("validateBondAssetInput returned error: %v", err)
	}
	if repoInput.CouponFrequency != "semiannual" {
		t.Fatalf("CouponFrequency = %q, want semiannual", repoInput.CouponFrequency)
	}
}

func TestValidateBondAssetInputRejectsInvalidCouponFrequency(t *testing.T) {
	input := BondAssetInput{
		Symbol:          "ORI025",
		Name:            "ORI025",
		CouponFrequency: "weekly",
	}

	_, err := validateBondAssetInput(input)
	if !errors.Is(err, ErrInvalidBondCouponFrequency) {
		t.Fatalf("validateBondAssetInput error = %v, want ErrInvalidBondCouponFrequency", err)
	}
}

func TestValidateBondTransactionInputBuyWithAccruedCouponKeepsCleanCostBasis(t *testing.T) {
	input := BondTransactionInput{
		AccountID:           "account-id",
		AssetID:             "asset-id",
		TransactionType:     repository.BondTransactionBuy,
		PrincipalAmount:     1000000,
		Price:               1.002,
		AccruedCouponAmount: 21412,
	}

	repoInput, err := validateBondTransactionInput("portfolio-id", input)
	if err != nil {
		t.Fatalf("validateBondTransactionInput returned error: %v", err)
	}

	if repoInput.PrincipalAmount != 1000000 {
		t.Fatalf("PrincipalAmount = %v, want 1000000", repoInput.PrincipalAmount)
	}
	if repoInput.GrossAmount != 1002000 {
		t.Fatalf("GrossAmount = %v, want 1002000", repoInput.GrossAmount)
	}
	if repoInput.CostAmount != 1002000 {
		t.Fatalf("CostAmount = %v, want 1002000", repoInput.CostAmount)
	}
	if repoInput.AccruedCouponAmount != 21412 {
		t.Fatalf("AccruedCouponAmount = %v, want 21412", repoInput.AccruedCouponAmount)
	}
	if repoInput.NetAmount != 1023412 {
		t.Fatalf("NetAmount = %v, want 1023412", repoInput.NetAmount)
	}
}

func TestValidateBondTransactionInputRejectsNegativeAccruedCoupon(t *testing.T) {
	input := BondTransactionInput{
		AccountID:           "account-id",
		AssetID:             "asset-id",
		TransactionType:     repository.BondTransactionBuy,
		PrincipalAmount:     1000000,
		AccruedCouponAmount: -1,
	}

	_, err := validateBondTransactionInput("portfolio-id", input)
	if !errors.Is(err, ErrInvalidBondAmount) {
		t.Fatalf("validateBondTransactionInput error = %v, want ErrInvalidBondAmount", err)
	}
}

func TestValidateBondTransactionInputDefaultsAccruedCouponToZeroForCoupon(t *testing.T) {
	input := BondTransactionInput{
		AccountID:           "account-id",
		AssetID:             "asset-id",
		TransactionType:     repository.BondTransactionCoupon,
		GrossAmount:         50000,
		AccruedCouponAmount: 21412,
	}

	repoInput, err := validateBondTransactionInput("portfolio-id", input)
	if err != nil {
		t.Fatalf("validateBondTransactionInput returned error: %v", err)
	}
	if repoInput.AccruedCouponAmount != 0 {
		t.Fatalf("AccruedCouponAmount = %v, want 0", repoInput.AccruedCouponAmount)
	}
	if repoInput.NetAmount != 50000 {
		t.Fatalf("NetAmount = %v, want 50000", repoInput.NetAmount)
	}
}

func TestValidateBondInputIncludesAccruedCouponAndFeeInInitialBuy(t *testing.T) {
	input := BondInput{
		BondAssetInput:      BondAssetInput{Symbol: "ORI025", Name: "ORI025"},
		AccountID:           "account-id",
		PrincipalAmount:     1000000,
		CostAmount:          1002000,
		AccruedCouponAmount: 21412,
		FeeAmount:           1000,
	}

	repoInput, err := validateBondInput(input)
	if err != nil {
		t.Fatalf("validateBondInput returned error: %v", err)
	}
	if repoInput.CostAmount != 1002000 {
		t.Fatalf("CostAmount = %v, want 1002000", repoInput.CostAmount)
	}
	if repoInput.AccruedCouponAmount != 21412 {
		t.Fatalf("AccruedCouponAmount = %v, want 21412", repoInput.AccruedCouponAmount)
	}
	if repoInput.FeeAmount != 1000 {
		t.Fatalf("FeeAmount = %v, want 1000", repoInput.FeeAmount)
	}
}
