package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	AppName         = "ncm-cli"
	envConfigDir    = "NCM_CONFIG_DIR"
	sessionFileMode = 0o600
	dirMode         = 0o700
)

type Paths struct {
	ConfigDir        string
	SessionDir       string
	ProfileDir       string
	StorageStatePath string
	UserPath         string
}

type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

type StorageState struct {
	Cookies []Cookie `json:"cookies"`
	Origins []Origin `json:"origins"`
}

type Origin struct {
	Origin       string             `json:"origin"`
	LocalStorage []LocalStorageItem `json:"localStorage"`
}

type LocalStorageItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type UserInfo struct {
	ExportedAt        string `json:"exportedAt"`
	UserID            int64  `json:"userId"`
	Nickname          string `json:"nickname"`
	CSRFPresent       bool   `json:"csrfPresent"`
	StorageStatePath  string `json:"storageStatePath"`
	SourceProfileDir  string `json:"sourceProfileDir"`
	CompatStoragePath string `json:"compatStoragePath,omitempty"`
}

func Resolve(configDir string) (Paths, error) {
	dir, err := resolveConfigDir(configDir)
	if err != nil {
		return Paths{}, err
	}
	sessionDir := filepath.Join(dir, "session")
	return Paths{
		ConfigDir:        dir,
		SessionDir:       sessionDir,
		ProfileDir:       filepath.Join(dir, "chrome-profile"),
		StorageStatePath: filepath.Join(sessionDir, "storage-state.json"),
		UserPath:         filepath.Join(sessionDir, "user.json"),
	}, nil
}

func resolveConfigDir(configDir string) (string, error) {
	if configDir == "" {
		configDir = os.Getenv(envConfigDir)
	}
	if configDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("获取用户配置目录失败: %w", err)
		}
		configDir = filepath.Join(base, AppName)
	}
	return filepath.Abs(configDir)
}

func EnsureDirs(paths Paths) error {
	for _, dir := range []string{paths.ConfigDir, paths.SessionDir, paths.ProfileDir} {
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
	}
	return nil
}

func ExistingStorageState(primary Paths) (string, bool, error) {
	if fileExists(primary.StorageStatePath) {
		return primary.StorageStatePath, false, nil
	}
	compat := filepath.Join(".ncm", "session", "storage-state.json")
	absCompat, err := filepath.Abs(compat)
	if err != nil {
		return "", false, err
	}
	if fileExists(absCompat) {
		return absCompat, true, nil
	}
	return "", false, fmt.Errorf("未找到登录态，请先执行 ncm login")
}

func LoadStorageState(path string) (*StorageState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取登录态失败: %w", err)
	}
	var state StorageState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("解析登录态失败: %w", err)
	}
	if len(state.Cookies) == 0 {
		return nil, errors.New("登录态中没有 cookie")
	}
	return &state, nil
}

func WriteUserInfo(path string, info UserInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, sessionFileMode); err != nil {
		return err
	}
	return os.Chmod(path, sessionFileMode)
}

func HardenSessionFile(path string) error {
	if !fileExists(path) {
		return nil
	}
	return os.Chmod(path, sessionFileMode)
}

func CSRF(state *StorageState) string {
	for _, cookie := range state.Cookies {
		if cookie.Name == "__csrf" {
			return cookie.Value
		}
	}
	return ""
}

func HasCookie(state *StorageState, name string) bool {
	for _, cookie := range state.Cookies {
		if cookie.Name == name && cookie.Value != "" {
			return true
		}
	}
	return false
}

func CookiesForJar(state *StorageState, rawURL string) ([]*http.Cookie, *url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, nil, err
	}
	cookies := make([]*http.Cookie, 0, len(state.Cookies))
	for _, c := range state.Cookies {
		if c.Name == "" {
			continue
		}
		cookie := &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Domain:   strings.TrimPrefix(c.Domain, "."),
			HttpOnly: c.HTTPOnly,
			Secure:   c.Secure,
		}
		if cookie.Path == "" {
			cookie.Path = "/"
		}
		if c.Expires > 0 {
			cookie.Expires = time.Unix(int64(c.Expires), 0)
		}
		cookies = append(cookies, cookie)
	}
	return cookies, u, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
