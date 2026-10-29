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

// downloadTimeout 单次下载超时。慢网（如国内访问 GitHub）可能需要 10+ 分钟。
const downloadTimeout = 15 * time.Minute

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

// DownloadURLs 返回某版本 pandoc 的多个下载地址，按优先级排。
// 中国大陆访问 GitHub 不稳定，优先走镜像代理。
func DownloadURLs(version string) ([]string, string, error) {
	if version == "" {
		version = DefaultVersion
	}
	p, err := platformFor(version)
	if err != nil {
		return nil, "", err
	}
	ghPath := fmt.Sprintf("jgm/pandoc/releases/download/%s/%s", version, p.urlSuffix)
	urls := []string{
		"https://ghfast.top/https://github.com/" + ghPath,
		"https://gh-proxy.com/https://github.com/" + ghPath,
		"https://github.com/" + ghPath,
	}
	return urls, p.archive, nil
}

// DownloadURL 返回首选下载地址（兼容旧调用）。
func DownloadURL(version string) (string, string, error) {
	urls, archive, err := DownloadURLs(version)
	if err != nil {
		return "", "", err
	}
	return urls[0], archive, nil
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
// 依次尝试镜像地址，任一成功即可。
func Install(ctx context.Context, version string, progress func(string)) (string, error) {
	if version == "" {
		version = DefaultVersion
	}
	urls, archive, err := DownloadURLs(version)
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

	var tmpPath string
	var lastErr error
	for i, url := range urls {
		log(fmt.Sprintf("[%d/%d] 尝试下载：%s", i+1, len(urls), url))
		path, err := downloadToTemp(ctx, url, archive, log)
		if err != nil {
			log(fmt.Sprintf("    失败：%v", err))
			lastErr = err
			continue
		}
		tmpPath = path
		break
	}
	if tmpPath == "" {
		return "", fmt.Errorf("所有下载地址均失败（最后一个错误：%v）。\n请检查网络，或手动安装：\n  1. winget install JohnMacFarlane.Pandoc\n  2. 或从 https://pandoc.org/installing.html 下载后设置 PANDOC_PATH 环境变量", lastErr)
	}
	defer os.Remove(tmpPath)

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

// downloadToTemp 下载 url 到临时文件，返回路径。
func downloadToTemp(ctx context.Context, url, archive string, log func(string)) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "paperagent/pandoc-installer")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	suffix := "." + archive
	tmp, err := os.CreateTemp("", "pandoc-download-*"+suffix)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()

	if resp.ContentLength > 0 {
		log(fmt.Sprintf("    总大小 %d MB", resp.ContentLength/1024/1024))
	}
	pr := &progressReader{
		r:     resp.Body,
		total: resp.ContentLength,
		log:   log,
	}
	if _, err := io.Copy(tmp, pr); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("写入失败: %w", err)
	}
	tmp.Close()
	return tmpPath, nil
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
			p.log(fmt.Sprintf("    已下载 %d / %d MB", p.read/1024/1024, p.total/1024/1024))
		} else {
			p.log(fmt.Sprintf("    已下载 %d MB", p.read/1024/1024))
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
