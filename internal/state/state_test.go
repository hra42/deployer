package state

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestServiceJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 5, 7, 12, 30, 0, 0, time.UTC)
	in := Service{
		Repo:       "owner/name",
		Slug:       "app-example-com",
		ClonePath:  "/srv/deployer/app-example-com",
		DeployedAt: ts,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), "2026-05-07T12:30:00Z") {
		t.Fatalf("expected RFC3339 timestamp in %s", b)
	}
	var out Service
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.DeployedAt.Equal(in.DeployedAt) {
		t.Fatalf("deployed_at: got %v want %v", out.DeployedAt, in.DeployedAt)
	}
	out.DeployedAt = in.DeployedAt
	if out != in {
		t.Fatalf("round-trip: got %+v want %+v", out, in)
	}
}

func TestStateJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)
	in := State{
		"a.example.com": {Repo: "owner/a", Slug: "a-example-com", ClonePath: "/srv/x/a-example-com", DeployedAt: ts},
		"b.example.com": {Repo: "owner/b", Slug: "b-example-com", ClonePath: "/srv/x/b-example-com", DeployedAt: ts},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out State
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != len(in) {
		t.Fatalf("len: got %d want %d", len(out), len(in))
	}
	for k, v := range in {
		got, ok := out[k]
		if !ok {
			t.Fatalf("missing key %q", k)
		}
		if !got.DeployedAt.Equal(v.DeployedAt) || got.Repo != v.Repo || got.Slug != v.Slug || got.ClonePath != v.ClonePath {
			t.Fatalf("entry %q: got %+v want %+v", k, got, v)
		}
	}
}

func TestShellQuote(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "''"},
		{"plain", "abc", "'abc'"},
		{"spaces", "a b", "'a b'"},
		{"single quote", "o'malley", `'o'\''malley'`},
		{"dollar", "$HOME", "'$HOME'"},
		{"backtick", "`x`", "'`x`'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shellQuote(tc.in)
			if got != tc.want {
				t.Fatalf("shellQuote(%q): got %q want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestPaths(t *testing.T) {
	s, l := paths("/srv/x")
	if s != "/srv/x/.deployer-state.json" {
		t.Errorf("state path: %q", s)
	}
	if l != "/srv/x/.deployer-state.lock" {
		t.Errorf("lock path: %q", l)
	}
}

func TestBuildLoadCmd(t *testing.T) {
	cmd := buildLoadCmd("/srv/x")
	for _, sub := range []string{
		"flock -s",
		"'/srv/x/.deployer-state.json'",
		"'/srv/x/.deployer-state.lock'",
		"echo '\\''{}'\\''",
		"sh -c",
	} {
		if !strings.Contains(cmd, sub) {
			t.Errorf("missing %q in: %s", sub, cmd)
		}
	}
}

func TestBuildUpsertCmd(t *testing.T) {
	ts := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	svc := Service{
		Repo:       "owner/name",
		Slug:       "ex-com",
		ClonePath:  "/srv/x/ex-com",
		DeployedAt: ts,
	}
	cmd, err := buildUpsertCmd("/srv/x", "ex.com", svc)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for _, sub := range []string{
		"flock -x",
		"python3 -c",
		"'/srv/x/.deployer-state.json'",
		"'/srv/x/.deployer-state.lock'",
		"'ex.com'",
	} {
		if !strings.Contains(cmd, sub) {
			t.Errorf("missing %q in: %s", sub, cmd)
		}
	}

	// Locate the patch-JSON arg (the only JSON-shaped quoted token).
	patchJSON, ok := extractQuotedToken(cmd, `{"ex.com":`)
	if !ok {
		t.Fatalf("could not find patch JSON in: %s", cmd)
	}
	var got map[string]Service
	if err := json.Unmarshal([]byte(patchJSON), &got); err != nil {
		t.Fatalf("patch JSON did not parse (%v): %s", err, patchJSON)
	}
	if len(got) != 1 {
		t.Fatalf("patch should have 1 entry, got %d", len(got))
	}
	gotSvc, ok := got["ex.com"]
	if !ok {
		t.Fatalf("patch missing 'ex.com' key: %v", got)
	}
	if gotSvc.Repo != svc.Repo || gotSvc.Slug != svc.Slug || gotSvc.ClonePath != svc.ClonePath || !gotSvc.DeployedAt.Equal(svc.DeployedAt) {
		t.Fatalf("svc round-trip mismatch: got %+v want %+v", gotSvc, svc)
	}
}

func TestBuildUpsertCmdEmptyDomain(t *testing.T) {
	_, err := buildUpsertCmd("/srv/x", "", Service{})
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
}

func TestBuildUpsertCmdQuoteSafety(t *testing.T) {
	svc := Service{
		Repo:       "owner/name",
		Slug:       "x",
		ClonePath:  "/srv/has space/x",
		DeployedAt: time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
	}
	cmd, err := buildUpsertCmd("/srv/has space/x", "o'malley.com", svc)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// shellQuote always wraps in single quotes; any literal ' becomes '\''.
	// That sequence has 2 lone quotes (the closer of the prior segment and the
	// opener of the next). Across the whole command, every ' must pair up.
	if strings.Count(cmd, "'")%2 != 0 {
		t.Fatalf("unbalanced single quotes in: %s", cmd)
	}
	// The escaped form for o'malley.com must appear.
	if !strings.Contains(cmd, `'o'\''malley.com'`) {
		t.Errorf("expected escaped domain in: %s", cmd)
	}
	// And the path with a space must appear quoted.
	if !strings.Contains(cmd, `'/srv/has space/x/.deployer-state.json'`) {
		t.Errorf("expected quoted spaced state path in: %s", cmd)
	}
}

// extractQuotedToken finds a single-quoted token in cmd that, after unquoting,
// starts with prefix. Returns the unquoted token. Handles the '\'' escape
// produced by shellQuote.
func extractQuotedToken(cmd, prefix string) (string, bool) {
	// shellQuote produces tokens of the form 'X' where any ' in X becomes '\''.
	// To recover X, find every maximal '...'-wrapped run separated by '\''
	// markers. Simpler approach: walk the string, collecting characters that
	// would survive shell-unquoting, and search for the prefix.
	for i := 0; i < len(cmd); i++ {
		if cmd[i] != '\'' {
			continue
		}
		// Try to parse a shell-quoted token starting here.
		token, end, ok := readQuoted(cmd, i)
		if !ok {
			continue
		}
		if strings.HasPrefix(token, prefix) {
			return token, true
		}
		i = end
	}
	return "", false
}

// readQuoted parses a shell single-quoted token starting at cmd[i] (which is
// '\''). It handles the '\'' escape. Returns the unquoted content, the index
// of the closing quote, and ok=true on success.
func readQuoted(cmd string, i int) (string, int, bool) {
	if i >= len(cmd) || cmd[i] != '\'' {
		return "", 0, false
	}
	var b strings.Builder
	i++ // skip opening '
	for i < len(cmd) {
		if cmd[i] == '\'' {
			// Either end of token, or start of a '\'' escape sequence.
			if i+3 < len(cmd) && cmd[i+1] == '\\' && cmd[i+2] == '\'' && cmd[i+3] == '\'' {
				b.WriteByte('\'')
				i += 4
				continue
			}
			return b.String(), i, true
		}
		b.WriteByte(cmd[i])
		i++
	}
	return "", 0, false
}
