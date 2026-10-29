package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemp 写临时文件到 t.TempDir()，返回路径。
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

// captureStdout 临时重定向 os.Stdout，捕获 run 的打印输出。
func captureStdout(t *testing.T, f func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		done <- string(out)
	}()

	runErr := f()
	w.Close()
	os.Stdout = old
	captured := <-done
	return captured, runErr
}

const minimalDesign = `{
  "topic": "测试主题",
  "data_source": "CSMAR",
  "sample": "A 股制造业",
  "dependent_var": {"name": "创新", "measure": "专利数"},
  "core_var": {"name": "数字化", "measure": "词频"},
  "model": "双向固定效应"
}`

const minimalRefs = `[
  {
    "type": "article-journal",
    "authors": ["张三"],
    "title": "测试文献一",
    "container": "测试期刊",
    "year": 2023,
    "lang": "zh"
  },
  {
    "type": "article-journal",
    "authors": ["李四"],
    "title": "测试文献二",
    "container": "测试期刊",
    "year": 2022,
    "lang": "zh"
  }
]`

func TestRun_Minimal(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	out := filepath.Join(t.TempDir(), "draft.md")

	err := run([]string{
		"--topic", "测试主题",
		"--mode", "thesis",
		"--output", out,
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output not found: %v", err)
	}
	md := string(data)

	// 至少有标题
	if !strings.Contains(md, "# 测试主题") {
		t.Errorf("markdown 缺少标题")
	}
	// 至少有 3 个章节标题（Mock 环境下章节结构固定）
	sectionCount := strings.Count(md, "\n## ")
	if sectionCount < 3 {
		t.Errorf("章节数过少: %d", sectionCount)
	}
}

func TestRun_WithDesign(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	designPath := writeTemp(t, "design.json", minimalDesign)
	out := filepath.Join(t.TempDir(), "draft.md")

	err := run([]string{
		"--mode", "journal",
		"--design", designPath,
		"--output", out,
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	data, _ := os.ReadFile(out)
	md := string(data)

	// 设计输入里的 topic 应被采用
	if !strings.Contains(md, "测试主题") {
		t.Errorf("标题未使用 design 的 topic")
	}
	// 研究设计节应有内容（模板生成）
	if !strings.Contains(md, "## 研究设计") {
		t.Errorf("缺少研究设计节")
	}
	if !strings.Contains(md, "CSMAR") {
		t.Errorf("研究设计节未包含 design 的数据来源")
	}
}

func TestRun_WithRefs(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	refsPath := writeTemp(t, "refs.json", minimalRefs)
	out := filepath.Join(t.TempDir(), "draft.md")

	err := run([]string{
		"--topic", "测试主题",
		"--mode", "thesis",
		"--refs", refsPath,
		"--output", out,
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	data, _ := os.ReadFile(out)
	md := string(data)

	if !strings.Contains(md, "## 参考文献") {
		t.Errorf("缺少参考文献节")
	}
	if !strings.Contains(md, "测试文献一") {
		t.Errorf("参考文献未包含第一条")
	}
	if !strings.Contains(md, "[1]") {
		t.Errorf("参考文献未编号")
	}
}

func TestRun_InvalidMode(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	out := filepath.Join(t.TempDir(), "draft.md")

	err := run([]string{
		"--topic", "测试",
		"--mode", "invalid",
		"--output", out,
	})
	if err == nil {
		t.Fatal("无效 mode 应返回错误")
	}
	if !strings.Contains(err.Error(), "invalid --mode") {
		t.Errorf("错误信息不符: %v", err)
	}
}

func TestRun_DesignNotFound(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	out := filepath.Join(t.TempDir(), "draft.md")

	err := run([]string{
		"--mode", "thesis",
		"--design", "E:/nonexistent/design.json",
		"--output", out,
	})
	if err == nil {
		t.Fatal("design 文件不存在应返回错误")
	}
}

// TestRun_QualityReport 验证 run 结束后打印质量报告标记。
func TestRun_QualityReport(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	refsPath := writeTemp(t, "refs.json", minimalRefs)
	out := filepath.Join(t.TempDir(), "draft.md")

	stdout, err := captureStdout(t, func() error {
		return run([]string{
			"--topic", "测试主题",
			"--mode", "thesis",
			"--refs", refsPath,
			"--output", out,
		})
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if !strings.Contains(stdout, "QUALITY_STATUS=") {
		t.Errorf("stdout 缺 QUALITY_STATUS 标记，实际输出：\n%s", stdout)
	}
	if !strings.Contains(stdout, "QUALITY_PLACEHOLDERS=") {
		t.Errorf("stdout 缺 QUALITY_PLACEHOLDERS 标记，实际输出：\n%s", stdout)
	}
}

// TestRun_OutputSubstantial 防止"跑通但输出空壳"。
func TestRun_OutputSubstantial(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	out := filepath.Join(t.TempDir(), "draft.md")

	err := run([]string{
		"--topic", "测试主题",
		"--mode", "thesis",
		"--output", out,
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output not found: %v", err)
	}
	md := string(data)

	if len(md) < 2000 {
		t.Errorf("输出过短：%d 字节，怀疑空壳", len(md))
	}
	if strings.Contains(md, "（本节未生成）") {
		t.Errorf("输出含「（本节未生成）」空壳标记")
	}
	sectionCount := strings.Count(md, "\n## ")
	if sectionCount < 2 {
		t.Errorf("章节数过少：%d", sectionCount)
	}
}

// TestRun_InstallPandocFlagAccepted 验证 --install-pandoc=false 被解析且不触发下载。
func TestRun_InstallPandocFlagAccepted(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	out := filepath.Join(t.TempDir(), "draft.md")
	err := run([]string{
		"--topic", "测试",
		"--mode", "thesis",
		"--output", out,
		"--install-pandoc=false",
	})
	if err != nil {
		t.Fatalf("--install-pandoc=false 不应报错: %v", err)
	}
}
