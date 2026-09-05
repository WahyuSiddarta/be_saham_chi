package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type failedBondRepository struct{ BondRepository }

func (failedBondRepository) GetBond(context.Context, string, string, string) (repository.PortfolioBond, error) {
	return repository.PortfolioBond{AssetID: "partial"}, repository.ErrPortfolioNotFound
}

func (failedBondRepository) ListBonds(context.Context, string, string) ([]repository.PortfolioBond, error) {
	return []repository.PortfolioBond{{AssetID: "partial"}}, repository.ErrPortfolioNotFound
}

func TestBondReadsPreserveErrorCauseAndDiscardPartialResults(t *testing.T) {
	s := NewBondService(failedBondRepository{})
	bond, err := s.GetBond(context.Background(), "user", "portfolio", "asset")
	if !reflect.DeepEqual(bond, repository.PortfolioBond{}) || !errors.Is(err, repository.ErrPortfolioNotFound) {
		t.Fatalf("bond=%+v error=%v", bond, err)
	}
	if err.Error() != "portfolioService.GetBond -> PortfolioRepository.GetBond: portfolio not found" {
		t.Fatalf("error message changed: %v", err)
	}
	bonds, err := s.ListBonds(context.Background(), "user", "portfolio")
	if bonds != nil || !errors.Is(err, repository.ErrPortfolioNotFound) {
		t.Fatalf("bonds=%+v error=%v", bonds, err)
	}
	if err.Error() != "portfolioService.ListBonds -> PortfolioRepository.ListBonds: portfolio not found" {
		t.Fatalf("error message changed: %v", err)
	}
}
