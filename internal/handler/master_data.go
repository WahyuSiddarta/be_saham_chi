package handler

import (
	"errors"
	"net/http"

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
		h.logRequestError(req, http.StatusInternalServerError, "failed to list master data", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to list master data")
	}

	data := make([]MasterDataResponse, 0, len(items))
	for _, item := range items {
		data = append(data, newMasterDataResponse(item))
	}
	return response.Success(w, http.StatusOK, data)
}

func (h Handler) UpdateMasterData(w http.ResponseWriter, req *http.Request) error {
	var request UpdateMasterDataRequest
	if err := binding.BindJSON(req.Body, &request); err != nil || request.Value == nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	}

	item, err := h.masterDataService.UpdateMasterData(req.Context(), chi.URLParam(req, "key"), *request.Value)
	if err != nil {
		if errors.Is(err, service.ErrInvalidMasterDataKey) || errors.Is(err, service.ErrInvalidMasterDataValue) {
			h.logRequestError(req, http.StatusBadRequest, err.Error(), err)
			return response.Fail(w, http.StatusBadRequest, err.Error())
		}
		h.logRequestError(req, http.StatusInternalServerError, "failed to update master data", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to update master data")
	}

	data := newMasterDataResponse(item)
	return response.Success(w, http.StatusOK, data)
}
