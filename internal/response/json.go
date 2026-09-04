package response

import (
	"encoding/json"
	"net/http"
)

type body struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

type failureBody struct {
	Status string `json:"status"`
	Data   *any   `json:"data,omitempty"`
}

// Success writes a successful JSON response.
func Success(w http.ResponseWriter, status int, data any) error {
	return writeJSON(w, status, body{
		Status: "ok",
		Data:   data,
	})
}

// Fail writes an unsuccessful JSON response.
func Fail(w http.ResponseWriter, status int, data any) error {
	response := failureBody{
		Status: "nok",
	}
	if data != nil {
		response.Data = &data
	}

	return writeJSON(w, status, response)
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(data)
}
