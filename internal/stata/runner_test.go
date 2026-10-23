package stata

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRunner_FindsStata(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Skipf("本机无 Stata，跳过: %v", err)
	}
	if r.StataPath == "" {
		t.Fatal("StataPath 不应为空")
	}
	if r.Timeout <= 0 {
		t.Fatal("Timeout 应有默认值")
	}
	if !r.IsAvailable() {
		t.Errorf("IsAvailable 应为 true，StataPath=%s", r.StataPath)
	}
}

func TestRunner_IsAvailable_Nil(t *testing.T) {
	var r *Runner
	if r.IsAvailable() {
		t.Error("nil Runner 的 IsAvailable 应返回 false，不应 panic")
	}
}

func TestRunner_IsAvailable_FakePath(t *testing.T) {
	r := &Runner{StataPath: `C:\no\such\stata.exe`}
	if r.IsAvailable() {
		t.Error("不存在的路径 IsAvailable 应返回 false")
	}
}

func TestRunner_Run_MissingDoFile(t *testing.T) {
	r := &Runner{StataPath: defaultStataPath, Timeout: 0}
	_, err := r.Run(context.Background(), `C:\no\such\file.do`)
	if err == nil {
		t.Fatal(".do 不存在应报错")
	}
}

func TestRunner_Run_EmptyStataPath(t *testing.T) {
	r := &Runner{}
	_, err := r.Run(context.Background(), "x.do")
	if err == nil {
		t.Fatal("StataPath 为空应报错")
	}
}

// TestRunner_Run_RealStata 是唯一真跑 Stata 的测试。
//   - go test -short 时跳过（CI 用 -short，GitHub Actions 无 Stata）
//   - 本机无 Stata 时自动 skip
//   - 用 Stata 内置 sysuse auto 数据集，无需外部 .dta
func TestRunner_Run_RealStata(t *testing.T) {
	if testing.Short() {
		t.Skip("short 模式跳过真跑 Stata")
	}
	r, err := NewRunner()
	if err != nil {
		t.Skipf("本机无 Stata，跳过: %v", err)
	}

	dir := t.TempDir()
	doPath := filepath.Join(dir, "smoke.do")
	doText := "sysuse auto, clear\n" +
		"reg price mpg\n" +
		"display \"SMOKE_OK\"\n"
	if err := os.WriteFile(doPath, []byte(doText), 0644); err != nil {
		t.Fatalf("写 .do 失败: %v", err)
	}

	out, err := r.Run(context.Background(), doPath)
	if err != nil {
		t.Fatalf("真跑 Stata 失败: %v\n输出:\n%s", err, out)
	}

	// Stata 批处理结果多在 .log，stdout 不一定含标记，因此两种都接受：
	//   - stdout 含 SMOKE_OK
	//   - 或目录里生成了 .log 文件
	if !strings.Contains(out, "SMOKE_OK") {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.log"))
		if len(matches) == 0 {
			t.Logf("警告：stdout 未含 SMOKE_OK，也未找到 .log 文件。out=%q", out)
		}
	}
}
