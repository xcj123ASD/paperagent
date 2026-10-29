package export

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Exporter 把 Markdown 导出为可交付文件格式。
type Exporter interface {
	// Export 读取 mdPath，写出 outputPath。
	Export(ctx context.Context, mdPath, outputPath string) error
	Name() string
}

// PandocExporter 通过 pandoc 命令行把 Markdown 转为 docx / tex。
//
// pandoc 路径解析顺序：
//  1. PandocExporter.Binary（显式指定，测试用）
//  2. 环境变量 PANDOC_PATH（便携版推荐）
//  3. 用户目录 ~/.paperagent/pandoc/（首启自动下载安装位置）
//  4. PATH 中的 pandoc（系统安装）
type PandocExporter struct {
	// Binary 为空时自动解析
	Binary  string
	Timeout time.Duration
}

func NewPandocExporter() *PandocExporter {
	return &PandocExporter{
		Timeout: 60 * time.Second,
	}
}

func (p *PandocExporter) Name() string { return "pandoc" }

// UserPandocDir 返回用户目录下的 pandoc 安装目录：~/.paperagent/pandoc/
func UserPandocDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	return filepath.Join(home, ".paperagent", "pandoc"), nil
}

// UserPandocBinary 返回用户目录下 pandoc 可执行文件路径。
func UserPandocBinary() (string, error) {
	dir, err := UserPandocDir()
	if err != nil {
		return "", err
	}
	name := "pandoc"
	if runtime.GOOS == "windows" {
		name = "pandoc.exe"
	}
	return filepath.Join(dir, name), nil
}

// resolveBinary 返回 pandoc 可执行文件路径，找不到时返回错误。
func (p *PandocExporter) resolveBinary() (string, error) {
	if p.Binary != "" {
		if _, err := os.Stat(p.Binary); err != nil {
			return "", fmt.Errorf("PANDOC_PATH 指向的文件不存在: %s", p.Binary)
		}
		return p.Binary, nil
	}

	if envPath := os.Getenv("PANDOC_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err != nil {
			return "", fmt.Errorf("PANDOC_PATH 指向的文件不存在: %s", envPath)
		}
		return envPath, nil
	}

	// 用户目录（首启自动下载位置）
	if userBin, err := UserPandocBinary(); err == nil {
		if _, statErr := os.Stat(userBin); statErr == nil {
			return userBin, nil
		}
	}

	bin, err := exec.LookPath("pandoc")
	if err != nil {
		return "", fmt.Errorf("未找到 pandoc：运行 paperagent --install-pandoc 自动下载，或设置 PANDOC_PATH 环境变量，或 winget install JohnMacFarlane.Pandoc")
	}
	return bin, nil
}

func (p *PandocExporter) Export(ctx context.Context, mdPath, outputPath string) error {
	if _, err := os.Stat(mdPath); err != nil {
		return fmt.Errorf("输入文件不存在: %s", mdPath)
	}

	bin, err := p.resolveBinary()
	if err != nil {
		return err
	}

	// 按扩展名选输出格式
	ext := strings.ToLower(filepath.Ext(outputPath))
	var pandocFormat string
	switch ext {
	case ".docx":
		pandocFormat = "docx"
	case ".tex":
		pandocFormat = "latex"
	case ".html":
		pandocFormat = "html"
	default:
		return fmt.Errorf("不支持的输出格式: %s（支持 .docx / .tex / .html）", ext)
	}

	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{
		mdPath,
		"-o", outputPath,
		"-f", "markdown",
		"-t", pandocFormat,
		"--standalone",
	}

	cmd := exec.CommandContext(cctx, bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if cctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("pandoc 执行超时（%v）", timeout)
		}
		return fmt.Errorf("pandoc 执行失败: %w\n输出: %s", err, strings.TrimSpace(string(out)))
	}

	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("pandoc 声称成功但输出文件不存在: %s", outputPath)
	}
	return nil
}
