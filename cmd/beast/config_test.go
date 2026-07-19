package main

import "testing"

func TestGenerateConfigSecret(t *testing.T) {
	first, err := generateConfigSecret(32)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generateConfigSecret(32)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) < 32 || first == second {
		t.Fatalf("generated secrets are not sufficiently distinct: %q %q", first, second)
	}
}
