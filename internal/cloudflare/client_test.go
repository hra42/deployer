package cloudflare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("auth header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":{"hello":"world"}}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	out, err := c.do(context.Background(), "GET", "/anything", nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(string(out), `"hello":"world"`) {
		t.Errorf("result body = %s", out)
	}
}

func TestDoErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1004,"message":"DNS Validation Error"}],"result":null}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	_, err := c.do(context.Background(), "GET", "/x", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "DNS Validation Error") {
		t.Errorf("err = %v", err)
	}
}

func TestDoNon2xxNoEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"success":false,"errors":[],"result":null}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	_, err := c.do(context.Background(), "GET", "/x", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("err = %v", err)
	}
}
