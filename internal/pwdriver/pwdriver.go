package pwdriver

import (
	"fmt"
	"io"

	"github.com/playwright-community/playwright-go"
)

// Install downloads the Playwright driver. When withBrowser is true it also
// installs Playwright Chromium for machines without a system Chrome.
func Install(withBrowser bool, stdout io.Writer, stderr io.Writer) error {
	options := &playwright.RunOptions{
		SkipInstallBrowsers: !withBrowser,
		Stdout:              stdout,
		Stderr:              stderr,
	}
	if withBrowser {
		options.Browsers = []string{"chromium"}
	}
	return playwright.Install(options)
}

// Run starts Playwright and automatically prepares the driver if it is missing.
func Run(stdout io.Writer, stderr io.Writer) (*playwright.Playwright, error) {
	pw, err := playwright.Run()
	if err == nil {
		return pw, nil
	}

	if stdout != nil {
		fmt.Fprintln(stdout, "正在准备 Playwright driver...")
	}
	installErr := Install(false, stdout, stderr)
	if installErr != nil {
		return nil, fmt.Errorf("启动 Playwright 失败，自动安装 driver 也失败: %v；原始错误: %w", installErr, err)
	}

	pw, err = playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("启动 Playwright 失败，已尝试自动安装 driver: %w", err)
	}
	return pw, nil
}
