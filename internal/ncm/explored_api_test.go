package ncm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ncmcrypto "ncm-cli/internal/crypto"
)

func TestPlayerURLPayload(t *testing.T) {
	path, payload := capturePayload(t, `{"code":200,"data":[]}`, func(ctx context.Context, client *Client) error {
		_, err := client.PlayerURL(ctx, 210049, "lossless")
		return err
	})
	if path != "/weapi/song/enhance/player/url/v1" {
		t.Fatalf("path = %q", path)
	}
	if payload["ids"] != "[210049]" {
		t.Fatalf("ids = %v", payload["ids"])
	}
	if payload["level"] != "lossless" {
		t.Fatalf("level = %v", payload["level"])
	}
	if payload["encodeType"] != "aac" {
		t.Fatalf("encodeType = %v", payload["encodeType"])
	}
}

func TestRecommendSongsResponseShapes(t *testing.T) {
	_, _ = capturePayload(t, `{"code":200,"recommend":[{"id":1,"name":"from-recommend"}]}`, func(ctx context.Context, client *Client) error {
		res, err := client.RecommendSongs(ctx)
		if err != nil {
			return err
		}
		songs := res.Songs()
		if len(songs) != 1 || songs[0].Name != "from-recommend" {
			t.Fatalf("recommend songs = %#v", songs)
		}
		return nil
	})

	path, _ := capturePayload(t, `{"code":200,"data":{"dailySongs":[{"id":2,"name":"from-daily"}]}}`, func(ctx context.Context, client *Client) error {
		res, err := client.RecommendSongs(ctx)
		if err != nil {
			return err
		}
		songs := res.Songs()
		if len(songs) != 1 || songs[0].Name != "from-daily" {
			t.Fatalf("daily songs = %#v", songs)
		}
		return nil
	})
	if path != "/weapi/v2/discovery/recommend/songs" {
		t.Fatalf("path = %q", path)
	}
}

func TestPlayRecordPayload(t *testing.T) {
	path, payload := capturePayload(t, `{"code":200,"weekData":[],"allData":[]}`, func(ctx context.Context, client *Client) error {
		_, err := client.PlayRecord(ctx, 12345, -1)
		return err
	})
	if path != "/weapi/v1/play/record" {
		t.Fatalf("path = %q", path)
	}
	if payload["uid"].(float64) != 12345 {
		t.Fatalf("uid = %v", payload["uid"])
	}
	if payload["type"].(float64) != -1 {
		t.Fatalf("type = %v", payload["type"])
	}
}

func capturePayload(t *testing.T, response string, call func(context.Context, *Client) error) (string, map[string]any) {
	t.Helper()
	var gotPath string
	var gotPlain map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("params") != "encrypted" || r.PostForm.Get("encSecKey") != "key" {
			t.Fatalf("form = %s", r.PostForm.Encode())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		HTTP:    server.Client(),
		CSRF:    "csrf-value",
		Encrypt: func(data []byte) (ncmcrypto.WeAPIForm, error) {
			if err := json.Unmarshal(data, &gotPlain); err != nil {
				t.Fatal(err)
			}
			return ncmcrypto.WeAPIForm{Params: "encrypted", EncSecKey: "key"}, nil
		},
	}
	if err := call(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	return gotPath, gotPlain
}
