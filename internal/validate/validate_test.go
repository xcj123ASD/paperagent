package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSize_SmallFile_OK(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "small.json")
	if err := os.WriteFile(p, []byte("{}"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := FileSize(p, 0); err != nil {
		t.Errorf("小文件应通过，实际: %v", err)
	}
}

func TestFileSize_TooLarge_Error(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "big.json")
	big := make([]byte, 1024)
	if err := os.WriteFile(p, big, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	err := FileSize(p, 100)
	if err == nil {
		t.Fatal("超大文件应报错")
	}
	if !strings.Contains(err.Error(), "文件过大") {
		t.Errorf("错误信息应含「文件过大」，实际: %v", err)
	}
}

func TestFileSize_NotFound_Error(t *testing.T) {
	err := FileSize("E:/nonexistent/file.json", 0)
	if err == nil {
		t.Fatal("不存在的文件应报错")
	}
}

func TestFileSize_Directory_Error(t *testing.T) {
	dir := t.TempDir()
	err := FileSize(dir, 0)
	if err == nil {
		t.Fatal("目录应报错")
	}
	if !strings.Contains(err.Error(), "目录") {
		t.Errorf("错误信息应含「目录」，实际: %v", err)
	}
}

func TestFileSize_DefaultLimit(t *testing.T) {
	// limitBytes=0 时用 DefaultMaxBytes
	dir := t.TempDir()
	p := filepath.Join(dir, "x.json")
	if err := os.WriteFile(p, []byte("{}"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := FileSize(p, 0); err != nil {
		t.Errorf("默认限制下小文件应通过，实际: %v", err)
	}
}
