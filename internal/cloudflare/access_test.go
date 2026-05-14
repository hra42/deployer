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

func TestFindAccessAppNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/accounts/acc1/access/apps") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "domain=example.com") {
			t.Errorf("missing domain in query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":[]}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	app, err := c.FindAccessApp(context.Background(), "acc1", "example.com")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if app != nil {
		t.Errorf("expected nil app, got %+v", app)
	}
}

func TestFindAccessAppReturnsFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":[
			{"id":"app1","name":"example.com","domain":"example.com","type":"self_hosted"}
		]}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	app, err := c.FindAccessApp(context.Background(), "acc1", "example.com")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if app == nil || app.ID != "app1" || app.Domain != "example.com" {
		t.Errorf("got %+v", app)
	}
}

func TestCreateAccessAppBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/accounts/acc1/access/apps") {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got struct {
			Name     string   `json:"name"`
			Domain   string   `json:"domain"`
			Type     string   `json:"type"`
			Policies []string `json:"policies"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got.Type != "self_hosted" || got.Name != "example.com" || got.Domain != "example.com" {
			t.Errorf("body = %+v", got)
		}
		if len(got.Policies) != 1 || got.Policies[0] != "pol1" {
			t.Errorf("policies = %+v, want [pol1]", got.Policies)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":{"id":"app1","name":"example.com","domain":"example.com","type":"self_hosted"}}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	app, err := c.CreateAccessApp(context.Background(), "acc1", "example.com", "example.com", []string{"pol1"}, AccessAppOptions{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if app == nil || app.ID != "app1" {
		t.Errorf("got %+v", app)
	}
}

func TestUpdateAccessAppPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/accounts/acc1/access/apps/app1") {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got struct {
			Policies []string `json:"policies"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(got.Policies) != 1 || got.Policies[0] != "pol1" {
			t.Errorf("policies = %+v, want [pol1]", got.Policies)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":{"id":"app1"}}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	if err := c.UpdateAccessApp(context.Background(), "acc1", "app1", "example.com", "example.com", []string{"pol1"}, AccessAppOptions{}); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestCreateAccessAppInstantAuthBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var got struct {
			AllowedIdPs            []string `json:"allowed_idps"`
			AutoRedirectToIdentity bool     `json:"auto_redirect_to_identity"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(got.AllowedIdPs) != 1 || got.AllowedIdPs[0] != "idp1" {
			t.Errorf("allowed_idps = %+v", got.AllowedIdPs)
		}
		if !got.AutoRedirectToIdentity {
			t.Errorf("auto_redirect_to_identity = false, want true")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":{"id":"app1"}}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	_, err := c.CreateAccessApp(context.Background(), "acc1", "example.com", "example.com", []string{"pol1"},
		AccessAppOptions{AllowedIdPs: []string{"idp1"}, AutoRedirectToIdentity: true})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestListIdentityProviders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/accounts/acc1/access/identity_providers") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"errors":[],"result":[
			{"id":"idp1","name":"EntraID","type":"azureAD"},
			{"id":"idp2","name":"","type":"onetimepin"}
		]}`))
	}))
	defer srv.Close()

	c := newWithBase("tok", srv.URL)
	idps, err := c.ListIdentityProviders(context.Background(), "acc1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(idps) != 2 || idps[0].Type != "azureAD" || idps[1].Type != "onetimepin" {
		t.Fatalf("got %+v", idps)
	}
}
