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
