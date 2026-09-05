package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
)

type UpdateMasterDataRequest struct {
	Value *float64 `json:"value"`
}

type MasterDataResponse struct {
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
	UpdatedAt string  `json:"updated_at"`
}

func (h Handler) ListMasterData(w http.ResponseWriter, req *http.Request) error {
	items, err := h.masterDataService.ListMasterData(req.Context())
	if err != nil {
		return newHTTPError(http.StatusInternalServerError, "failed to list master data").SetInternal(err)
	}

	data := make([]MasterDataResponse, 0, len(items))
	for _, item := range items {
		data = append(data, newMasterDataResponse(item))
	}
	return response.JSON(w, http.StatusOK, map[string]any{"status": "ok", "data": data})
}

func (h Handler) UpdateMasterData(w http.ResponseWriter, req *http.Request) error {
	var request UpdateMasterDataRequest
	if err := binding.BindJSON(req.Body, &request); err != nil || request.Value == nil {
		return newHTTPError(http.StatusBadRequest, "invalid request body")
	}

	item, err := h.masterDataService.UpdateMasterData(req.Context(), chi.URLParam(req, "key"), *request.Value)
	if err != nil {
		if errors.Is(err, service.ErrInvalidMasterDataKey) || errors.Is(err, service.ErrInvalidMasterDataValue) {
			return newHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		return newHTTPError(http.StatusInternalServerError, "failed to update master data").SetInternal(err)
	}

	data := newMasterDataResponse(item)
	return response.JSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"key":        data.Key,
		"value":      data.Value,
		"updated_at": data.UpdatedAt,
	})
}

func newMasterDataResponse(item repository.MasterData) MasterDataResponse {
	return MasterDataResponse{
		Key:       item.Key,
		Value:     item.Value,
		UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
