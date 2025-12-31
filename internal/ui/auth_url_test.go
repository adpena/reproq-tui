package ui

import "testing"

func TestNormalizeDjangoURLInput(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "example.com", want: "https://example.com"},
		{in: "localhost:8000", want: "http://localhost:8000"},
		{in: "127.0.0.1:9100", want: "http://127.0.0.1:9100"},
		{in: "192.168.1.5:8000", want: "http://192.168.1.5:8000"},
		{in: "https://example.com/reproq/stats/", want: "https://example.com"},
		{in: "https:\\example.com\\reproq\\stats\\", want: "https://example.com"},
	}

	for _, c := range cases {
		got, err := normalizeDjangoURLInput(c.in)
		if err != nil {
			t.Fatalf("normalize %q failed: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("normalize %q => %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeDjangoURLInputInvalid(t *testing.T) {
	if _, err := normalizeDjangoURLInput(" "); err == nil {
		t.Fatal("expected error for empty input")
	}
	if _, err := normalizeDjangoURLInput("ftp://example.com"); err == nil {
		t.Fatal("expected error for invalid scheme")
	}
}
