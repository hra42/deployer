package slug

import "testing"

func TestDomain(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"app.example.com", "app-example-com"},
		{"App.Example.COM", "app-example-com"},
		{"  app.example.com  ", "app-example-com"},
		{"single", "single"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := Domain(tc.in); got != tc.want {
			t.Errorf("Domain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
