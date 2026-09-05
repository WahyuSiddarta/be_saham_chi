package request

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestBindJSON(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"object", `{"name":"main"}`, true},
		{"unknown fields", `{"name":"main","extra":true}`, true},
		{"whitespace", " \n" + `{"name":"main"}` + " \n", true},
		{"empty", "", false},
		{"malformed", `{"name":`, false},
		{"wrong field type", `{"name":12}`, false},
		{"multiple values", `{"name":"one"} {"name":"two"}`, false},
		{"trailing garbage", `{"name":"one"} junk`, false},
		{"duplicate keys", `{"name":"one","name":"two"}`, false},
		{"oversized", `{"name":"` + strings.Repeat("x", int(MaxJSONBodyBytes)) + `"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var dst struct {
				Name string `json:"name"`
			}
			err := BindJSON(strings.NewReader(tc.body), &dst)
			if (err == nil) != tc.valid {
				t.Fatalf("error=%v valid=%v", err, tc.valid)
			}
			if tc.valid && dst.Name != "main" {
				t.Fatalf("name=%q", dst.Name)
			}
		})
	}
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }
func TestBindJSONReaderErrorsAndInvalidDestination(t *testing.T) {
	want := errors.New("read failed")
	var dst struct{}
	if err := BindJSON(failingReader{want}, &dst); !errors.Is(err, want) {
		t.Fatalf("error=%v", err)
	}
	if err := BindJSON(nil, &dst); err == nil {
		t.Fatal("nil body accepted")
	}
	if err := BindJSON(strings.NewReader("{}"), dst); err == nil {
		t.Fatal("non-pointer destination accepted")
	}
	if err := BindJSON(io.LimitReader(strings.NewReader("{}"), 1), &dst); err == nil {
		t.Fatal("truncated body accepted")
	}
}
