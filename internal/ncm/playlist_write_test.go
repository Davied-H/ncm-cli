package ncm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ncmcrypto "ncm-cli/internal/crypto"
)

func TestPlaylistCreatePayloads(t *testing.T) {
	_, payload := capturePlaylistWritePayload(t, func(ctx context.Context, client *Client) error {
		_, err := client.PlaylistCreate(ctx, "公开歌单", false, "token-value")
		return err
	})
	if payload["name"] != "公开歌单" {
		t.Fatalf("name = %v", payload["name"])
	}
	if payload["checkToken"] != "token-value" {
		t.Fatalf("checkToken = %v", payload["checkToken"])
	}
	if _, ok := payload["privacy"]; ok {
		t.Fatalf("public create should omit privacy, got %v", payload["privacy"])
	}

	path, payload := capturePlaylistWritePayload(t, func(ctx context.Context, client *Client) error {
		_, err := client.PlaylistCreate(ctx, "私密歌单", true, "token-value")
		return err
	})
	if path != "/weapi/playlist/create" {
		t.Fatalf("path = %q", path)
	}
	if payload["privacy"].(float64) != 10 {
		t.Fatalf("privacy = %v", payload["privacy"])
	}
}

func TestPlaylistManipulateTracksPayloads(t *testing.T) {
	path, payload := capturePlaylistWritePayload(t, func(ctx context.Context, client *Client) error {
		_, err := client.PlaylistAddTracks(ctx, 123, []int64{1, 2})
		return err
	})
	if path != "/weapi/playlist/manipulate/tracks" {
		t.Fatalf("path = %q", path)
	}
	if payload["op"] != "add" || payload["trackIds"] != "[1,2]" || payload["imme"] != true {
		t.Fatalf("add payload = %#v", payload)
	}
	if payload["pid"].(float64) != 123 {
		t.Fatalf("pid = %v", payload["pid"])
	}

	_, payload = capturePlaylistWritePayload(t, func(ctx context.Context, client *Client) error {
		_, err := client.PlaylistRemoveTracks(ctx, 123, []int64{3})
		return err
	})
	if payload["op"] != "del" || payload["trackIds"] != "[3]" {
		t.Fatalf("remove payload = %#v", payload)
	}
}

func TestPlaylistMetadataPayloads(t *testing.T) {
	tests := []struct {
		name   string
		call   func(context.Context, *Client) error
		assert func(*testing.T, string, map[string]any)
	}{
		{
			name: "rename",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.PlaylistRename(ctx, 123, "new-name")
				return err
			},
			assert: func(t *testing.T, path string, payload map[string]any) {
				if path != "/weapi/playlist/update/name" || payload["name"] != "new-name" {
					t.Fatalf("rename path=%q payload=%#v", path, payload)
				}
			},
		},
		{
			name: "tags",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.PlaylistUpdateTags(ctx, 123, []string{"华语", "流行"})
				return err
			},
			assert: func(t *testing.T, path string, payload map[string]any) {
				if path != "/weapi/playlist/tags/update" || payload["tags"] != "华语,流行" {
					t.Fatalf("tags path=%q payload=%#v", path, payload)
				}
			},
		},
		{
			name: "description",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.PlaylistUpdateDescription(ctx, 123, "desc", "token-value")
				return err
			},
			assert: func(t *testing.T, path string, payload map[string]any) {
				if path != "/weapi/playlist/desc/update" || payload["desc"] != "desc" || payload["checkToken"] != "token-value" {
					t.Fatalf("desc path=%q payload=%#v", path, payload)
				}
			},
		},
		{
			name: "delete",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.PlaylistDelete(ctx, 123)
				return err
			},
			assert: func(t *testing.T, path string, payload map[string]any) {
				if path != "/weapi/playlist/delete" || payload["pid"].(float64) != 123 {
					t.Fatalf("delete path=%q payload=%#v", path, payload)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, payload := capturePlaylistWritePayload(t, tt.call)
			tt.assert(t, path, payload)
		})
	}
}

func capturePlaylistWritePayload(t *testing.T, call func(context.Context, *Client) error) (string, map[string]any) {
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
		_, _ = w.Write([]byte(`{"code":200,"playlist":{"id":99,"name":"created"}}`))
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
