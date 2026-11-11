package crm

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestCRMBrandStaticFiles(t *testing.T) {
    for _, tc := range []struct{ path, mime, snippet string }{
        {"/crm/favicon.svg", "image/svg+xml", "<svg"},
        {"/crm/logo-dark.svg", "image/svg+xml", "svg"},
        {"/crm/logo-light.svg", "image/svg+xml", "svg"},
        {"/crm/style.css", "text/css", "--brand-orange"},
    } {
        t.Run(tc.path, func(t *testing.T) {
            w := httptest.NewRecorder()
            (&Server{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
            if w.Code != http.StatusOK || !strings.Contains(w.Header().Get("Content-Type"), tc.mime) ||
                !strings.Contains(w.Body.String(), tc.snippet) {
                t.Fatalf("path %s: status %d, content-type %s", tc.path, w.Code, w.Header().Get("Content-Type"))
            }
        })
    }
}
