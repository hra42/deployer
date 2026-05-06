package sshx

import "testing"

func TestRedactCmd(t *testing.T) {
	in := `git clone https://x-access-token:ghp_secrettoken123@github.com/me/app /srv/app`
	got := redactCmd(in)
	want := `git clone https://x-access-token:***@github.com/me/app /srv/app`
	if got != want {
		t.Errorf("redactCmd:\n got: %s\nwant: %s", got, want)
	}
}

func TestParseHost(t *testing.T) {
	cases := []struct {
		in, user, addr string
		wantErr        bool
	}{
		{"root@1.2.3.4", "root", "1.2.3.4:22", false},
		{"deploy@example.com:2222", "deploy", "example.com:2222", false},
		{"noatsign", "", "", true},
		{"@host", "", "", true},
		{"user@", "", "", true},
	}
	for _, tc := range cases {
		u, a, err := parseHost(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("parseHost(%q) err=%v wantErr=%v", tc.in, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && (u != tc.user || a != tc.addr) {
			t.Errorf("parseHost(%q) = (%q, %q), want (%q, %q)", tc.in, u, a, tc.user, tc.addr)
		}
	}
}
