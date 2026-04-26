package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONDoesNotInventSensitiveFields(t *testing.T) {
	var buf bytes.Buffer
	value := map[string]any{
		"code": 200,
		"name": "playlist",
	}
	if err := JSON(&buf, value); err != nil {
		t.Fatal(err)
	}
	text := buf.String()
	for _, secret := range []string{"params", "encSecKey", "csrf_token", "MUSIC_U"} {
		if strings.Contains(text, secret) {
			t.Fatalf("output contains sensitive token %q: %s", secret, text)
		}
	}
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	if err := Table(&buf, []string{"ID", "NAME"}, [][]string{{"1", "周杰伦"}}); err != nil {
		t.Fatal(err)
	}
	text := buf.String()
	if !strings.Contains(text, "ID") || !strings.Contains(text, "周杰伦") {
		t.Fatalf("unexpected table output: %q", text)
	}
}

func TestTableAlignsWideCharacters(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{
		{"317107", "小岛", "杨千嬅"},
		{"2163629816", "STORIE BREVI", "Tananai/Annalisa"},
		{"718765", "ブルーバード", "いきものがかり"},
	}
	if err := Table(&buf, []string{"ID", "NAME", "ARTISTS"}, rows); err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"ID          NAME          ARTISTS",
		"317107      小岛          杨千嬅",
		"2163629816  STORIE BREVI  Tananai/Annalisa",
		"718765      ブルーバード  いきものがかり",
		"",
	}, "\n")
	if buf.String() != want {
		t.Fatalf("table output:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestDisplayWidth(t *testing.T) {
	tests := map[string]int{
		"abc":       3,
		"小岛":        4,
		"ブルーバード":    12,
		"Tananai/杏": 10,
	}
	for input, want := range tests {
		if got := displayWidth(input); got != want {
			t.Fatalf("displayWidth(%q) = %d, want %d", input, got, want)
		}
	}
}
