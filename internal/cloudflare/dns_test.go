package cloudflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFindCNAMENoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "type=CNAME") {
			t.Errorf("missing type=CNAME in query: %s", r.URL.RawQuery)
		}
		if !strings.Contains(r.URL.RawQuery, "name=example.com") {
			t.Errorf("missing name=example.com in query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":[]}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	rec, err := c.FindCNAME(context.Background(), "zoneA", "example.com")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rec != nil {
		t.Errorf("expected nil record, got %+v", rec)
	}
}

func TestFindCNAMEReturnsFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":[
			{"id":"rec1","type":"CNAME","name":"example.com","content":"old.target","proxied":true,"ttl":1}
		]}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	rec, err := c.FindCNAME(context.Background(), "zoneA", "example.com")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rec == nil || rec.ID != "rec1" || rec.Content != "old.target" {
		t.Errorf("got %+v", rec)
	}
}

func TestCreateCNAMEBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/zones/zoneA/dns_records") {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got DNSRecord
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got.Type != "CNAME" || got.Name != "example.com" || got.Content != "host.example.net" {
			t.Errorf("body = %+v", got)
		}
		if !got.Proxied {
			t.Errorf("expected proxied=true")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":{"id":"new1"}}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	if err := c.CreateCNAME(context.Background(), "zoneA", "example.com", "host.example.net"); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestUpdateCNAMEPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/zones/zoneA/dns_records/rec123") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":{"id":"rec123"}}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	if err := c.UpdateCNAME(context.Background(), "zoneA", "rec123", "example.com", "host.example.net"); err != nil {
		t.Fatalf("err: %v", err)
	}
}
