package ncm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"ncm-cli/internal/config"
	ncmcrypto "ncm-cli/internal/crypto"
)

const DefaultBaseURL = "https://music.163.com"

type EncryptFunc func([]byte) (ncmcrypto.WeAPIForm, error)

type Client struct {
	BaseURL string
	HTTP    *http.Client
	CSRF    string
	Encrypt EncryptFunc
}

type APIError struct {
	Code    int
	Message string
	Body    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("网易云 API 错误 code=%d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("网易云 API 错误 code=%d", e.Code)
}

func NewClientFromStorageState(state *config.StorageState, timeout time.Duration) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	cookies, u, err := config.CookiesForJar(state, DefaultBaseURL)
	if err != nil {
		return nil, err
	}
	jar.SetCookies(u, cookies)
	csrf := config.CSRF(state)
	if !config.HasCookie(state, "MUSIC_U") {
		return nil, errors.New("登录态缺少 MUSIC_U cookie，请重新执行 ncm login")
	}
	if csrf == "" {
		return nil, errors.New("登录态缺少 __csrf cookie，请重新执行 ncm login")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		BaseURL: DefaultBaseURL,
		HTTP: &http.Client{
			Jar: jar,
			Transport: &http.Transport{
				Proxy: nil,
			},
			Timeout: timeout,
		},
		CSRF:    csrf,
		Encrypt: ncmcrypto.EncryptWeAPI,
	}, nil
}

func (c *Client) WeAPI(ctx context.Context, path string, payload any, out any) error {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 30 * time.Second}
	}
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	encrypt := c.Encrypt
	if encrypt == nil {
		encrypt = ncmcrypto.EncryptWeAPI
	}
	bodyMap, err := payloadMap(payload)
	if err != nil {
		return err
	}
	if c.CSRF != "" {
		bodyMap["csrf_token"] = c.CSRF
	}
	plain, err := json.Marshal(bodyMap)
	if err != nil {
		return err
	}
	form, err := encrypt(plain)
	if err != nil {
		return err
	}
	endpoint, err := c.endpoint(path)
	if err != nil {
		return err
	}
	values := url.Values{}
	values.Set("csrf_token", c.CSRF)
	endpoint.RawQuery = values.Encode()
	post := url.Values{}
	post.Set("params", form.Params)
	post.Set("encSecKey", form.EncSecKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(post.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", c.BaseURL+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, safeBody(data))
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return errors.New("网易云 API 返回空响应")
	}
	if err := checkBusinessError(data); err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("解析响应 JSON 失败: %w", err)
	}
	return nil
}

func (c *Client) endpoint(path string) (*url.URL, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	normalized := NormalizePath(path)
	u := base.ResolveReference(&url.URL{Path: normalized})
	return u, nil
}

func NormalizePath(path string) string {
	if path == "" {
		return "/weapi"
	}
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if strings.HasPrefix(path, "/api/") {
		return "/weapi/" + strings.TrimPrefix(path, "/api/")
	}
	return path
}

func payloadMap(payload any) (map[string]any, error) {
	if payload == nil {
		return map[string]any{}, nil
	}
	if m, ok := payload.(map[string]any); ok {
		out := make(map[string]any, len(m)+1)
		for k, v := range m {
			out[k] = v
		}
		return out, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func checkBusinessError(data []byte) error {
	var meta struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("解析响应 JSON 失败: %w", err)
	}
	code, ok := codeAsInt(meta.Code)
	if !ok || code == 200 {
		return nil
	}
	message := meta.Message
	if message == "" {
		message = meta.Msg
	}
	return &APIError{Code: code, Message: message, Body: safeBody(data)}
}

func codeAsInt(value any) (int, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case float64:
		return int(v), true
	case string:
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n, true
		}
	}
	return 0, false
}

func safeBody(data []byte) string {
	text := strings.TrimSpace(string(data))
	if len(text) > 500 {
		return text[:500] + "..."
	}
	return text
}
