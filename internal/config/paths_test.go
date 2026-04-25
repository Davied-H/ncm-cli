package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePrecedence(t *testing.T) {
	t.Setenv(envConfigDir, filepath.Join(t.TempDir(), "env"))
	explicit := filepath.Join(t.TempDir(), "explicit")
	paths, err := Resolve(explicit)
	if err != nil {
		t.Fatal(err)
	}
	if paths.ConfigDir != explicit {
		t.Fatalf("ConfigDir = %q, want %q", paths.ConfigDir, explicit)
	}
	paths, err = Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if paths.ConfigDir != os.Getenv(envConfigDir) {
		t.Fatalf("ConfigDir = %q, want env dir", paths.ConfigDir)
	}
}

func TestLoadStorageStateAndCSRF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "storage-state.json")
	data := `{
		"cookies": [
			{"name":"MUSIC_U","value":"u","domain":".music.163.com","path":"/","expires":-1,"httpOnly":true,"secure":true},
			{"name":"__csrf","value":"csrf-value","domain":".music.163.com","path":"/","expires":-1,"httpOnly":false,"secure":false}
		],
		"origins": []
	}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := LoadStorageState(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := CSRF(state); got != "csrf-value" {
		t.Fatalf("CSRF = %q", got)
	}
	if !HasCookie(state, "MUSIC_U") {
		t.Fatal("MUSIC_U not found")
	}
	cookies, u, err := CookiesForJar(state, "https://music.163.com")
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "music.163.com" {
		t.Fatalf("host = %q", u.Host)
	}
	if len(cookies) != 2 {
		t.Fatalf("cookies len = %d", len(cookies))
	}
}
