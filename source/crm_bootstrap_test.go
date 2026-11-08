package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCRMBootstrapGuard(t *testing.T) {
	t.Setenv("LIIKKUVA_BOOTSTRAP", "1")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	h := liikkuvaSecurity(Config{}, next)

	tests := []struct {
		path string
		want int
	}{
		{"/crm/api/login", 204},
		{"/crm/api/jobs", 204},
		{"/lead", 503},
	}

	for _, tc := range tests {
		r := httptest.NewRequest(http.MethodPost, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.path, w.Code, tc.want)
		}
	}
}
