package export

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	p := NewPandocExporter()
	if got := p.Name(); got != "pandoc" {
		t.Errorf("Name() = %q，期望 pandoc", got)
	}
}

func TestNewPandocExporter_DefaultTimeout(t *testing.T) {
	p := NewPandocExporter()
	if p.Timeout <= 0 {
		t.Errorf("默认 Timeout 应 > 0，实际 %v", p.Timeout)
	}
}

// fakeBinary 造一个临时假 pandoc 文件（不是真 pandoc），
// 用于测 resolveBinary 的"文件存在"分支与 Export 的 exec 失败分支。
func fakeBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "fake-pandoc")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("写假 binary 失败: %v", err)
	}
	return p
}

// ---- Export: 前置检查 ----

func TestExport_MissingInput(t *testing.T) {
	p := NewPandocExporter()
	err := p.Export(context.Background(), `C:\no\such\file.md`, "out.docx")
	if err == nil {
		t.Fatal("源文件不存在应报错")
	}
	if !strings.Contains(err.Error(), "输入文件不存在") {
		t.Errorf("错误信息应含'输入文件不存在'，实际: %v", err)
	}
}

func TestExport_UnsupportedExt(t *testing.T) {
	bin := fakeBinary(t)
	mdPath := filepath.Join(t.TempDir(), "x.md")
	if err := os.WriteFile(mdPath, []byte("# t"), 0644); err != nil {
		t.Fatalf("写 md 失败: %v", err)
	}

	p := &PandocExporter{Binary: bin, Timeout: 5 * time.Second}
	err := p.Export(context.Background(), mdPath, "out.pdf")
	if err == nil {
		t.Fatal("不支持的扩展名应报错")
	}
	if !strings.Contains(err.Error(), "不支持的输出格式") {
		t.Errorf("错误信息应含'不支持的输出格式'，实际: %v", err)
	}
}

// ---- resolveBinary ----

func TestResolveBinary_BinarySet(t *testing.T) {
	bin := fakeBinary(t)
	p := &PandocExporter{Binary: bin}
	got, err := p.resolveBinary()
	if err != nil {
		t.Fatalf("resolveBinary 失败: %v", err)
	}
	if got != bin {
		t.Errorf("应返回 Binary 路径 %q，实际 %q", bin, got)
	}
}

func TestResolveBinary_BinaryMissing(t *testing.T) {
	p := &PandocExporter{Binary: `C:\no\such\pandoc.exe`}
	_, err := p.resolveBinary()
	if err == nil {
		t.Fatal("Binary 不存在应报错")
	}
	if !strings.Contains(err.Error(), "PANDOC_PATH") {
		t.Errorf("错误信息应含'PANDOC_PATH'，实际: %v", err)
	}
}

func TestResolveBinary_EnvPath(t *testing.T) {
	bin := fakeBinary(t)
	t.Setenv("PANDOC_PATH", bin)
	p := &PandocExporter{}
	got, err := p.resolveBinary()
	if err != nil {
		t.Fatalf("resolveBinary 失败: %v", err)
	}
	if got != bin {
		t.Errorf("应返回 PANDOC_PATH %q，实际 %q", bin, got)
	}
}

func TestResolveBinary_EnvPathMissing(t *testing.T) {
	t.Setenv("PANDOC_PATH", `C:\no\such\pandoc.exe`)
	p := &PandocExporter{}
	_, err := p.resolveBinary()
	if err == nil {
		t.Fatal("PANDOC_PATH 指向不存在应报错")
	}
	if !strings.Contains(err.Error(), "PANDOC_PATH") {
		t.Errorf("错误信息应含'PANDOC_PATH'，实际: %v", err)
	}
}

func TestResolveBinary_NotFound(t *testing.T) {
	// 清空 PANDOC_PATH 与 PATH，强制 LookPath 失败
	t.Setenv("PANDOC_PATH", "")
	t.Setenv("PATH", "")
	p := &PandocExporter{}
	_, err := p.resolveBinary()
	if err == nil {
		t.Fatal("PANDOC_PATH 空且 PATH 无 pandoc 应报错")
	}
	if !strings.Contains(err.Error(), "未找到 pandoc") {
		t.Errorf("错误信息应含'未找到 pandoc'，实际: %v", err)
	}
}

// ---- Export: 假 binary 执行失败（验证 exec 链路） ----

func TestExport_FakeBinaryExecFails(t *testing.T) {
	bin := fakeBinary(t)
	mdPath := filepath.Join(t.TempDir(), "x.md")
	if err := os.WriteFile(mdPath, []byte("# t"), 0644); err != nil {
		t.Fatalf("写 md 失败: %v", err)
	}

	p := &PandocExporter{Binary: bin, Timeout: 5 * time.Second}
	err := p.Export(context.Background(), mdPath, filepath.Join(t.TempDir(), "out.docx"))
	// 假 binary 不是真 pandoc，执行会失败或输出文件不存在，两种情况都应是 error
	if err == nil {
		t.Fatal("假 binary 执行应报错（既非真 pandoc，也产不出 .docx）")
	}
}

// ---- 真跑 Pandoc（-short 跳过；无 Pandoc 时自动 skip） ----

func TestExport_RealPandoc(t *testing.T) {
	if testing.Short() {
		t.Skip("short 模式跳过真跑 pandoc")
	}

	bin := os.Getenv("PANDOC_PATH")
	if bin == "" {
		var err error
		bin, err = exec.LookPath("pandoc")
		if err != nil {
			t.Skipf("本机无 pandoc，跳过: %v", err)
		}
	}

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "x.md")
	outPath := filepath.Join(dir, "x.docx")
	if err := os.WriteFile(mdPath, []byte("# 标题\n\n正文。\n"), 0644); err != nil {
		t.Fatalf("写 md 失败: %v", err)
	}

	p := &PandocExporter{Binary: bin, Timeout: 60 * time.Second}
	if err := p.Export(context.Background(), mdPath, outPath); err != nil {
		t.Fatalf("真跑 pandoc 失败: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("输出文件应存在: %v", err)
	}
}
