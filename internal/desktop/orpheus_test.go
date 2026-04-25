package desktop

import "testing"

func TestSongPlayURLMatchesWebClient(t *testing.T) {
	got, err := SongPlayURL(29816860)
	if err != nil {
		t.Fatalf("SongPlayURL returned error: %v", err)
	}
	want := "orpheus://eyJ0eXBlIjoic29uZyIsImlkIjoyOTgxNjg2MCwiY21kIjoicGxheSJ9"
	if got != want {
		t.Fatalf("SongPlayURL() = %q, want %q", got, want)
	}
}

func TestSongPlayURLRejectsInvalidID(t *testing.T) {
	if _, err := SongPlayURL(0); err == nil {
		t.Fatal("SongPlayURL(0) returned nil error")
	}
}
