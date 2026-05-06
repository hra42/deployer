package deploy

import (
	"strings"
	"testing"
)

func TestBuildCloneCmd(t *testing.T) {
	cases := []struct {
		name, repo string
		wantHost   string // expected host+path inside the URL
	}{
		{"https url", "https://github.com/me/app", "github.com/me/app"},
		{"https url with .git", "https://github.com/me/app.git", "github.com/me/app"},
		{"bare host path", "github.com/me/app", "github.com/me/app"},
		{"owner/name short", "me/app", "github.com/me/app"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := buildCloneCmd(tc.repo, "ghp_secret", "/srv/app-example-com", "/srv")
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			wantURL := "https://x-access-token:ghp_secret@" + tc.wantHost
			if !strings.Contains(cmd, wantURL) {
				t.Errorf("expected URL %q in command\n%s", wantURL, cmd)
			}
			if !strings.Contains(cmd, "mkdir -p '/srv'") {
				t.Errorf("expected mkdir for clone path:\n%s", cmd)
			}
			if !strings.Contains(cmd, "'/srv/app-example-com'") {
				t.Errorf("expected quoted remote path:\n%s", cmd)
			}
		})
	}
}

func TestBuildCloneCmdErrors(t *testing.T) {
	if _, err := buildCloneCmd("", "tok", "/a", "/b"); err == nil {
		t.Error("expected error on empty repo")
	}
	if _, err := buildCloneCmd("me/app", "", "/a", "/b"); err == nil {
		t.Error("expected error on empty token")
	}
	if _, err := buildCloneCmd("noslash", "tok", "/a", "/b"); err == nil {
		t.Error("expected error on malformed repo")
	}
}
