package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginRejectsMissingCaptchaToken(t *testing.T) {
	c := NewClient("test")
	r := httptest.NewRequest(http.MethodGet, "/login?provider=google&redirect_uri=https://app/callback", nil)
	w := httptest.NewRecorder()
	loginHandler(c).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
