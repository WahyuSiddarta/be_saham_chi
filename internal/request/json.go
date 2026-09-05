package request

import (
	"encoding/json/v2"
	"errors"
	"io"
)

const MaxJSONBodyBytes int64 = 1 << 20

// BindJSON decodes one JSON value from any reader; it requires no framework context.
// It accepts unknown fields to preserve the existing API's additive compatibility.
func BindJSON(body io.Reader, destination any) error {
	if body == nil {
		return errors.New("empty request body")
	}
	data, err := io.ReadAll(io.LimitReader(body, MaxJSONBodyBytes+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > MaxJSONBodyBytes {
		return errors.New("request body exceeds 1 MiB")
	}
	return json.Unmarshal(data, destination)
}
