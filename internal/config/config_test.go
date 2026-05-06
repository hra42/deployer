package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployer.yml")

	want := &Config{
		SSHHost:             "deploy@10.0.0.1",
		SSHKeyPath:          "/home/me/.ssh/id_ed25519",
		GitHubToken:         "ghp_secret",
		ClonePath:           "/srv/apps",
		TraefikNetwork:      "traefik_proxy",
		CloudflareAPIToken:  "cf_token",
		CloudflareZoneID:    "zone123",
		CloudflareAccountID: "acct456",
		ZeroTrustPolicyID:   "policy789",
		CNAMETarget:         DefaultCNAMETarget,
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("round-trip mismatch:\nwant %+v\ngot  %+v", want, got)
	}
}

func TestSaveFileMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployer.yml")
	if err := Save(path, &Config{}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("file perm = %o, want 0600", perm)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yml"))
	if err == nil {
		t.Fatal("expected error loading missing file")
	}
}
