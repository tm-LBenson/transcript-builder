package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverOneExplicitWins(t *testing.T) {
	explicit := fakeExecutable(t, "explicit.exe")
	env := fakeExecutable(t, "env.exe")
	local := fakeExecutable(t, "local.exe")
	t.Setenv("TEST_TOOL", env)

	got := discoverOne("missing.exe", explicit, "TEST_TOOL", []string{local})
	if got != explicit {
		t.Fatalf("explicit path should win, got %s", got)
	}
}

func TestDiscoverOneEnvBeforeAppLocal(t *testing.T) {
	env := fakeExecutable(t, "env.exe")
	local := fakeExecutable(t, "local.exe")
	t.Setenv("TEST_TOOL", env)

	got := discoverOne("missing.exe", "", "TEST_TOOL", []string{local})
	if got != env {
		t.Fatalf("env path should win, got %s", got)
	}
}

func TestDiscoverOneAppLocalBeforeMissingPath(t *testing.T) {
	local := fakeExecutable(t, "local.exe")
	got := discoverOne("missing.exe", "", "TEST_TOOL", []string{local})
	if got != local {
		t.Fatalf("app-local path should be used, got %s", got)
	}
}

func fakeExecutable(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("fake"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
