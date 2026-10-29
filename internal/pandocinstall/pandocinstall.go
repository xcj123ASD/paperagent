package pandocinstall

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"paperagent/internal/export"
)

// DefaultVersion 默认下载的 pandoc 版本。
const DefaultVersion = "3.1.11"

// downloadTimeout 下载整体超时。
const downloadTimeout = 10 * time.Minute

type platformInfo struct {
	urlSuffix string
	archive   string // "zip" 或 "tar.gz"
}

func platformFor(version string) (platformInfo, error) {
	switch runtime.GOOS {
	case "windows":
		return platformInfo{
			urlSuffix: fmt.Sprintf("pandoc-%s-windows-x86_64.zip", version),
			archive:   "zip",
		}, nil
	case "linux":
		return platformInfo{
			urlSuffix: fmt.Sprintf("pandoc-%s-linux-amd64.tar.gz", version),
			archive:   "tar.gz",
		}, nil
	case "darwin":
		return platformInfo{}, fmt.Errorf("macOS 请通过 brew 安装：brew install pandoc")
	default:
		return platformInfo{}, fmt.Errorf("不支持的平台：%s", runtime.GOOS)
	}
}

// DownloadURL 返回某版本 pandoc 在当前平台的下载地址与归档类型。
func DownloadURL(version string) (string, string, error) {
	if version == "" {
		version = DefaultVersion
	}
	p, err := platformFor(version)
	if err != nil {
		return "", "", err
	}
	url := fmt.Sprintf("https://github.com/jgm/pandoc/releases/download/%s/%s", version, p.urlSuffix)
	return url, p.archive, nil
}

// IsInstalled 检查用户目录是否已有 pandoc 可执行文件。
func IsInstalled() bool {
	bin, err := export.UserPandocBinary()
	if err != nil {
		return false
	}
	_, statErr := os.Stat(bin)
	return statErr == nil
}

// InstalledVersion 返回已安装 pandoc 的版本字符串，未安装或执行失败返回空。
func InstalledVersion(ctx context.Context) string {
	if !IsInstalled() {
		return ""
	}
	bin, err := export.UserPandocBinary()
	if err != nil {
		return ""
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, bin, "--version").Output()
	if err != nil {
		return ""
	}
	line := strings.SplitN(string(out), "\n", 2)[0]
	return strings.TrimSpace(line)
}

// Install 下载并安装 pandoc 到 ~/.paperagent/pandoc/。
// progress 可为 nil；非 nil 时回调下载进度文本。
// 返回安装后的可执行文件路径。
func Install(ctx context.Context, version string, progress func(string)) (string, error) {
	if version == "" {
		version = DefaultVersion
	}
	url, archive, err := DownloadURL(version)
	if err != nil {
		return "", err
	}
	log := func(msg string) {
		if progress != nil {
			progress(msg)
		}
	}

	destDir, err := export.UserPandocDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	log(fmt.Sprintf("下载地址：%s", url))
	cctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "paperagent/pandoc-installer")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败 HTTP %d: %s", resp.StatusCode, url)
	}

	suffix := "." + archive
	tmp, err := os.CreateTemp("", "pandoc-download-*"+suffix)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	log(fmt.Sprintf("下载中（总大小 %d MB）...", resp.ContentLength/1024/1024))
	pr := &progressReader{
		r:     resp.Body,
		total: resp.ContentLength,
		log:   log,
	}
	if _, err := io.Copy(tmp, pr); err != nil {
		tmp.Close()
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}
	tmp.Close()
	log("下载完成，开始解压")

	binName := "pandoc"
	if runtime.GOOS == "windows" {
		binName = "pandoc.exe"
	}
	destBin := filepath.Join(destDir, binName)

	switch archive {
	case "zip":
		err = extractZip(tmpPath, destBin, binName)
	case "tar.gz":
		err = extractTarGz(tmpPath, destBin, binName)
	default:
		err = fmt.Errorf("不支持的归档格式: %s", archive)
	}
	if err != nil {
		return "", err
	}

	if runtime.GOOS != "windows" {
		_ = os.Chmod(destBin, 0755)
	}

	log("校验中...")
	ver := InstalledVersion(ctx)
	if ver == "" {
		return "", fmt.Errorf("安装后 pandoc 无法执行：%s", destBin)
	}
	log(fmt.Sprintf("安装成功：%s", ver))
	return destBin, nil
}

// progressReader 每 5MB 回调一次进度。
type progressReader struct {
	r     io.Reader
	total int64
	read  int64
	last  int64
	log   func(string)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.read += int64(n)
	if p.read-p.last >= 5*1024*1024 {
		p.last = p.read
		if p.total > 0 {
			p.log(fmt.Sprintf("已下载 %d / %d MB", p.read/1024/1024, p.total/1024/1024))
		} else {
			p.log(fmt.Sprintf("已下载 %d MB", p.read/1024/1024))
		}
	}
	return n, err
}

func extractZip(zipPath, destBin, binName string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开 zip 失败: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) != binName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("读取 zip 条目失败: %w", err)
		}
		defer rc.Close()
		out, err := os.OpenFile(destBin, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("写入目标失败: %w", err)
		}
		defer out.Close()
		if _, err := io.Copy(out, rc); err != nil {
			return fmt.Errorf("解压写入失败: %w", err)
		}
		return nil
	}
	return fmt.Errorf("zip 中未找到 %s", binName)
}

func extractTarGz(tarPath, destBin, binName string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("打开归档失败: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip 解析失败: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar 读取失败: %w", err)
		}
		if filepath.Base(hdr.Name) != binName {
			continue
		}
		out, err := os.OpenFile(destBin, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("写入目标失败: %w", err)
		}
		defer out.Close()
		if _, err := io.Copy(out, tr); err != nil {
			return fmt.Errorf("解压写入失败: %w", err)
		}
		return nil
	}
	return fmt.Errorf("归档中未找到 %s", binName)
}
