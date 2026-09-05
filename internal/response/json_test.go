package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestResponseEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name  string
		write func(http.ResponseWriter, int, any) error
		code  int
		data  any
		want  string
	}{
		{"object", Success, 201, map[string]any{"id": "p-1"}, `{"status":"ok","data":{"id":"p-1"}}`},
		{"list", Success, 200, []string{"one", "two"}, `{"status":"ok","data":["one","two"]}`},
		{"empty success", Success, 200, nil, `{"status":"ok","data":null}`},
		{"failure message", Fail, 400, "invalid input", `{"status":"nok","data":"invalid input"}`},
		{"failure details", Fail, 400, map[string]any{"field": "name"}, `{"status":"nok","data":{"field":"name"}}`},
		{"empty failure", Fail, 500, nil, `{"status":"nok"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			if err := tc.write(res, tc.code, tc.data); err != nil {
				t.Fatal(err)
			}
			if res.Code != tc.code {
				t.Fatalf("status=%d", res.Code)
			}
			if res.Header().Get("Content-Type") != "application/json" {
				t.Fatal("missing JSON content type")
			}
			var got, want any
			if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %s want %s", res.Body.String(), tc.want)
			}
		})
	}
}
