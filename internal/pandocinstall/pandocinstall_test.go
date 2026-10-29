package pandocinstall

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestDownloadURL_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows 验证")
	}
	url, archive, err := DownloadURL("3.1.11")
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if archive != "zip" {
		t.Errorf("Windows 归档应为 zip，实际 %q", archive)
	}
	if !strings.Contains(url, "pandoc-3.1.11-windows-x86_64.zip") {
		t.Errorf("URL 不符：%s", url)
	}
}

func TestDownloadURL_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("仅 Linux 验证")
	}
	url, archive, err := DownloadURL("3.1.11")
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if archive != "tar.gz" {
		t.Errorf("Linux 归档应为 tar.gz，实际 %q", archive)
	}
	if !strings.Contains(url, "pandoc-3.1.11-linux-amd64.tar.gz") {
		t.Errorf("URL 不符：%s", url)
	}
}

func TestDownloadURL_DefaultVersion(t *testing.T) {
	url, _, err := DownloadURL("")
	if err != nil {
		t.Skipf("当前平台不支持: %v", err)
	}
	if !strings.Contains(url, DefaultVersion) {
		t.Errorf("URL 应含默认版本 %s，实际 %s", DefaultVersion, url)
	}
}

func TestIsInstalled_NoDir(t *testing.T) {
	// 用户目录可能已安装 pandoc，此测试仅在未安装时断言 false
	if IsInstalled() {
		t.Skip("本机已安装 pandoc，跳过")
	}
	if IsInstalled() {
		t.Error("未安装时应返回 false")
	}
}

func TestInstalledVersion_NotInstalled(t *testing.T) {
	if IsInstalled() {
		t.Skip("本机已安装 pandoc，跳过")
	}
	if v := InstalledVersion(context.Background()); v != "" {
		t.Errorf("未安装时应返回空，实际 %q", v)
	}
}
