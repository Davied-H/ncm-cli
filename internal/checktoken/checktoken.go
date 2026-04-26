package checktoken

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"

	"ncm-cli/internal/pwdriver"
)

const defaultTimeout = 30 * time.Second

func Get(ctx context.Context, profileDir string, storageStatePath string, timeout time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if profileDir == "" && storageStatePath == "" {
		return "", fmt.Errorf("缺少登录态路径，请先执行 ncm login")
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	timeoutMS := float64(timeout.Milliseconds())

	pw, err := pwdriver.Run(os.Stdout, os.Stderr)
	if err != nil {
		return "", fmt.Errorf("启动 Playwright 失败，请先安装 driver：ncm driver install: %w", err)
	}
	defer pw.Stop()

	var browser playwright.Browser
	var browserCtx playwright.BrowserContext
	useProfile := profileAvailable(profileDir)
	if !useProfile && storageStatePath == "" {
		return "", fmt.Errorf("缺少登录态路径，请先执行 ncm login")
	}
	if useProfile {
		browserCtx, err = pw.Chromium.LaunchPersistentContext(profileDir, playwright.BrowserTypeLaunchPersistentContextOptions{
			Channel:           playwright.String("chrome"),
			Headless:          playwright.Bool(false),
			Viewport:          &playwright.Size{Width: 1280, Height: 900},
			Locale:            playwright.String("zh-CN"),
			TimezoneId:        playwright.String("Asia/Shanghai"),
			IgnoreHttpsErrors: playwright.Bool(true),
			Args:              []string{"--no-proxy-server"},
			Timeout:           playwright.Float(timeoutMS),
			UserAgent:         playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
		})
	} else {
		browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
			Channel:  playwright.String("chrome"),
			Headless: playwright.Bool(false),
			Args:     []string{"--no-proxy-server"},
			Timeout:  playwright.Float(timeoutMS),
		})
		if err == nil {
			defer browser.Close()
			browserCtx, err = browser.NewContext(playwright.BrowserNewContextOptions{
				StorageStatePath:  playwright.String(storageStatePath),
				Viewport:          &playwright.Size{Width: 1280, Height: 900},
				Locale:            playwright.String("zh-CN"),
				TimezoneId:        playwright.String("Asia/Shanghai"),
				IgnoreHttpsErrors: playwright.Bool(true),
				UserAgent:         playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
			})
		}
	}
	if err != nil {
		return "", fmt.Errorf("启动 Chrome 获取 checkToken 失败，请确认已安装 Chrome 或 Playwright Chromium: %w", err)
	}
	defer browserCtx.Close()

	pages := browserCtx.Pages()
	var page playwright.Page
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = browserCtx.NewPage()
		if err != nil {
			return "", err
		}
	}
	page.SetDefaultTimeout(timeoutMS)
	if _, err := page.Goto("https://music.163.com/#/discover", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(timeoutMS),
	}); err != nil {
		return "", fmt.Errorf("打开网易云音乐获取 checkToken 失败: %w", err)
	}
	if _, err := page.WaitForFunction("() => window.NEJ?.P?.('nej.j')?.bc9T && window.GEnc === true", nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(timeoutMS),
	}); err != nil {
		return "", fmt.Errorf("等待网易云页面运行时失败: %w", err)
	}
	if _, err := page.WaitForFunction("() => typeof window.NEJ?.P?.('nm.x')?.lc1x === 'function'", nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(timeoutMS),
	}); err != nil {
		return "", fmt.Errorf("等待网易云 checkToken 运行时失败: %w", err)
	}
	page.WaitForTimeout(2500)
	value, err := page.Evaluate(`() => new Promise((resolve) => {
		const x = window.NEJ?.P?.('nm.x');
		if (!x || typeof x.lc1x !== 'function') return resolve('');
		let done = false;
		let attempts = 0;
		const timer = setTimeout(() => {
			if (!done) {
				done = true;
				resolve('');
			}
		}, 8000);
		const attempt = () => {
			attempts += 1;
			x.lc1x((token) => {
				if (done) return;
				if (token || attempts >= 5) {
					done = true;
					clearTimeout(timer);
					resolve(token || '');
					return;
				}
				setTimeout(attempt, 1000);
			});
		};
		try {
			attempt();
		} catch {
			clearTimeout(timer);
			resolve('');
		}
	})`)
	if err != nil {
		return "", fmt.Errorf("读取 checkToken 失败: %w", err)
	}
	token, _ := value.(string)
	if token == "" {
		return "", fmt.Errorf("未能从网易云页面运行时获取 checkToken，请重新执行 ncm login 后重试")
	}
	return token, nil
}

func profileAvailable(profileDir string) bool {
	if profileDir == "" {
		return false
	}
	entries, err := os.ReadDir(profileDir)
	return err == nil && len(entries) > 0
}

func init() {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		_ = os.Unsetenv(key)
	}
}
