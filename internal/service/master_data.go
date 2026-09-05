package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

var (
	ErrInvalidMasterDataKey   = errors.New("invalid master data key")
	ErrInvalidMasterDataValue = errors.New("master data value must be greater than zero")
)

type masterDataRepository interface {
	ListMasterData(ctx context.Context) ([]repository.MasterData, error)
	UpdateMasterData(ctx context.Context, key string, value float64) (repository.MasterData, error)
}

type MasterDataService struct {
	repository masterDataRepository
}

func NewMasterDataService(repo masterDataRepository) *MasterDataService {
	return &MasterDataService{repository: repo}
}

func (s *MasterDataService) ListMasterData(ctx context.Context) ([]repository.MasterData, error) {
	items, err := s.repository.ListMasterData(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterDataService.List: %w", err)
	}
	return items, nil
}

func (s *MasterDataService) UpdateMasterData(ctx context.Context, key string, value float64) (repository.MasterData, error) {
	if key != "usd_idr_rate" && key != "bi_rate" {
		return repository.MasterData{}, ErrInvalidMasterDataKey
	}
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return repository.MasterData{}, ErrInvalidMasterDataValue
	}

	item, err := s.repository.UpdateMasterData(ctx, key, value)
	if err != nil {
		if errors.Is(err, repository.ErrMasterDataNotFound) {
			return repository.MasterData{}, ErrInvalidMasterDataKey
		}
		return repository.MasterData{}, fmt.Errorf("masterDataService.Update: %w", err)
	}
	return item, nil
}
