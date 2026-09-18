package service

import (
	"context"
	"testing"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type masterDataRepositoryStub struct {
	updatedKey   string
	updatedValue float64
}

func (s *masterDataRepositoryStub) ListMasterData(context.Context) ([]repository.MasterData, error) {
	return nil, nil
}

func (s *masterDataRepositoryStub) UpdateMasterData(_ context.Context, key string, value float64) (repository.MasterData, error) {
	s.updatedKey = key
	s.updatedValue = value
	return repository.MasterData{Key: key, Value: value}, nil
}

func TestMasterDataServiceUpdate(t *testing.T) {
	repoStub := &masterDataRepositoryStub{}
	service := NewMasterDataService(repoStub)

	item, err := service.UpdateMasterData(context.Background(), "usd_idr_rate", 16250)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if item.Value != 16250 || repoStub.updatedKey != "usd_idr_rate" {
		t.Fatalf("unexpected update: item=%+v key=%q", item, repoStub.updatedKey)
	}
}

func TestMasterDataServiceUpdateAllowsIndonesia10YearBondYield(t *testing.T) {
	repoStub := &masterDataRepositoryStub{}
	service := NewMasterDataService(repoStub)

	item, err := service.UpdateMasterData(context.Background(), MasterDataKeyIndonesia10YearBondYield, 6.75)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if item.Value != 6.75 || repoStub.updatedKey != MasterDataKeyIndonesia10YearBondYield {
		t.Fatalf("unexpected update: item=%+v key=%q", item, repoStub.updatedKey)
	}
}

func TestMasterDataServiceUpdateRejectsUnknownKey(t *testing.T) {
	service := NewMasterDataService(&masterDataRepositoryStub{})
	_, err := service.UpdateMasterData(context.Background(), "unknown", 1)
	if err != ErrInvalidMasterDataKey {
		t.Fatalf("error = %v, want %v", err, ErrInvalidMasterDataKey)
	}
}

func TestMasterDataServiceUpdateRejectsNonPositiveValue(t *testing.T) {
	service := NewMasterDataService(&masterDataRepositoryStub{})
	_, err := service.UpdateMasterData(context.Background(), MasterDataKeyIndonesia10YearBondYield, 0)
	if err != ErrInvalidMasterDataValue {
		t.Fatalf("error = %v, want %v", err, ErrInvalidMasterDataValue)
	}
}
