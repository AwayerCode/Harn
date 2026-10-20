package buildinfo

import "testing"

func TestResolvePrefersInjectedVersion(t *testing.T) {
	original := Version
	t.Cleanup(func() { Version = original })

	Version = "v0.2.0"
	if got := Resolve(); got != "v0.2.0" {
		t.Fatalf("got %q", got)
	}

	Version = "dev"
	if got := Resolve(); got == "" {
		t.Fatal("Resolve returned an empty version")
	}

	Version = ""
	if got := Resolve(); got == "" {
		t.Fatal("Resolve returned an empty version for an unset value")
	}
}
