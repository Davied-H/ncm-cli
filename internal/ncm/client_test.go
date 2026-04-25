package ncm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	ncmcrypto "ncm-cli/internal/crypto"
)

func TestNormalizePath(t *testing.T) {
	tests := map[string]string{
		"/api/user/playlist":       "/weapi/user/playlist",
		"api/user/playlist":        "/weapi/user/playlist",
		"/weapi/user/playlist":     "/weapi/user/playlist",
		"weapi/user/playlist":      "/weapi/user/playlist",
		"/api/w/nuser/account/get": "/weapi/w/nuser/account/get",
	}
	for input, want := range tests {
		if got := NormalizePath(input); got != want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestWeAPIInjectsCSRFAndPostsEncryptedForm(t *testing.T) {
	var gotPlain map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/weapi/user/playlist" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("csrf_token"); got != "csrf-value" {
			t.Fatalf("query csrf = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("params") != "encrypted" || r.PostForm.Get("encSecKey") != "key" {
			t.Fatalf("form = %s", r.PostForm.Encode())
		}
		if strings.Contains(r.PostForm.Encode(), "csrf-value") {
			t.Fatal("form leaked csrf")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"playlist":[]}`))
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
	var out PlaylistListResponse
	if err := client.WeAPI(context.Background(), "/api/user/playlist", map[string]any{"uid": 1}, &out); err != nil {
		t.Fatal(err)
	}
	if gotPlain["csrf_token"] != "csrf-value" {
		t.Fatalf("plain csrf = %v", gotPlain["csrf_token"])
	}
	if gotPlain["uid"].(float64) != 1 {
		t.Fatalf("uid = %v", gotPlain["uid"])
	}
}

func TestWeAPIBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":301,"message":"需要登录"}`))
	}))
	defer server.Close()
	client := testClient(server.URL)
	err := client.WeAPI(context.Background(), "/api/test", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type = %T", err)
	}
	if apiErr.Code != 301 {
		t.Fatalf("code = %d", apiErr.Code)
	}
}

func TestWeAPIInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()
	client := testClient(server.URL)
	err := client.WeAPI(context.Background(), "/api/test", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "JSON") {
		t.Fatalf("err = %v", err)
	}
}

func testClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 2 * time.Second,
		},
		CSRF: "csrf",
		Encrypt: func([]byte) (ncmcrypto.WeAPIForm, error) {
			return ncmcrypto.WeAPIForm{Params: url.QueryEscape("p"), EncSecKey: "k"}, nil
		},
	}
}
