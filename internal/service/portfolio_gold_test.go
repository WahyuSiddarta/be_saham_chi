package service

import (
	"errors"
	"testing"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

func TestValidateInitialGoldInputDefaultsAccountAndCreatesBuy(t *testing.T) {
	input, err := validateInitialGoldInput("portfolio-id", GoldInput{
		QuantityGrams: 5,
		Price:         1_750_000,
	})
	if err != nil {
		t.Fatalf("validate initial gold input: %v", err)
	}
	if input.AccountName != "Gold" {
		t.Fatalf("expected default account name Gold, got %q", input.AccountName)
	}
}

func TestValidateGoldTransactionStillRequiresExistingAccount(t *testing.T) {
	_, err := validateGoldInput("portfolio-id", GoldTransactionInput{
		TransactionType: repository.GoldTransactionBuy,
		QuantityGrams:   5,
		Price:           1_750_000,
	})
	if !errors.Is(err, ErrInvalidGoldAccount) {
		t.Fatalf("expected ErrInvalidGoldAccount, got %v", err)
	}
}
