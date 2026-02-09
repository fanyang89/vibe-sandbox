package main

import "testing"

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "Feature A", want: "feature-a"},
		{in: "a/b/c", want: "a-b-c"},
		{in: "_X-1.", want: "_x-1"},
		{in: "%%%", want: ""},
	}

	for _, tc := range cases {
		if got := normalizeName(tc.in); got != tc.want {
			t.Fatalf("normalizeName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidName(t *testing.T) {
	if !validName("ok-name_1.0") {
		t.Fatal("expected valid name")
	}
	if validName("bad/name") {
		t.Fatal("slash should be invalid")
	}
}

func TestGenerateName(t *testing.T) {
	name := generateName()
	if len(name) < len("sb-20060102-150405") {
		t.Fatalf("generated name too short: %q", name)
	}
}

func TestShortHashDeterministic(t *testing.T) {
	a := shortHash("same-input")
	b := shortHash("same-input")
	if a != b {
		t.Fatalf("shortHash should be deterministic, got %q vs %q", a, b)
	}
	if len(a) != 12 {
		t.Fatalf("shortHash length mismatch, got %d", len(a))
	}
}
