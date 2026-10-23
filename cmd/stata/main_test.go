package main

import (
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

func TestRun_HelpReturnsNil(t *testing.T) {
	if err := run([]string{"-h"}); err != nil {
		t.Fatalf("-h 应返回 nil，实际: %v", err)
	}
}

func TestRun_GeneratesDoOnly(t *testing.T) {
	outDo := filepath.Join(t.TempDir(), "out.do")
	err := run([]string{
		"-data", "testdata/fake.dta",
		"-dependent", "Innov",
		"-core", "Digital",
		"-controls", "Size,Lev",
		"-fixed-effects", "firm,year",
		"-cluster", "firm",
		"-model", "fe",
		"-label", "基准回归",
		"-output-do", outDo,
	})
	if err != nil {
		t.Fatalf("run 失败: %v", err)
	}

	data, err := os.ReadFile(outDo)
	if err != nil {
		t.Fatalf("读取 .do 失败: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"use \"testdata/fake.dta\"",
		"reg Innov Digital Size Lev i.firm i.year, cluster(firm)",
		"基准回归",
		"export delimited",
	} {
		if !strings.Contains(text, want) {
			t.Errorf(".do 缺少 %q\n实际:\n%s", want, text)
		}
	}

	// 未传 --run-stata 时不应生成 CSV
	csvPath := filepath.Join(filepath.Dir(outDo), "result.csv")
	if _, err := os.Stat(csvPath); err == nil {
		t.Errorf("未传 --run-stata 时不应生成 CSV，但 %s 存在", csvPath)
	}
}

func TestRun_MissingRequired(t *testing.T) {
	outDo := filepath.Join(t.TempDir(), "out.do")
	err := run([]string{
		"-core", "Digital",
		"-model", "fe",
		"-output-do", outDo,
	})
	if err == nil {
		t.Fatal("缺 data/dependent 应报错")
	}
	if _, statErr := os.Stat(outDo); statErr == nil {
		t.Errorf("必填缺失时不应生成 .do，但 %s 存在", outDo)
	}
}

func TestRun_InvalidModel(t *testing.T) {
	outDo := filepath.Join(t.TempDir(), "out.do")
	err := run([]string{
		"-data", "x.dta",
		"-dependent", "y",
		"-core", "x",
		"-model", "probit",
		"-output-do", outDo,
	})
	if err == nil {
		t.Fatal("不支持的模型应报错")
	}
	if !strings.Contains(err.Error(), "不支持的模型类型") {
		t.Errorf("错误信息应含'不支持的模型类型'，实际: %v", err)
	}
}

func TestRun_RequestJSON(t *testing.T) {
	outDo := filepath.Join(t.TempDir(), "out.do")
	reqJSON := `{
  "data_path": "testdata/from_json.dta",
  "output_path": "` + strings.ReplaceAll(filepath.Join(t.TempDir(), "result.csv"), `\`, `\\`) + `",
  "dependent": "Y",
  "core": "X",
  "controls": ["C1", "C2"],
  "fixed_effects": ["firm"],
  "cluster": "firm",
  "model": "fe",
  "label": "来自 JSON"
}`
	reqPath := writeTemp(t, "req.json", reqJSON)

	err := run([]string{
		"-request", reqPath,
		"-output-do", outDo,
	})
	if err != nil {
		t.Fatalf("run 失败: %v", err)
	}

	data, err := os.ReadFile(outDo)
	if err != nil {
		t.Fatalf("读取 .do 失败: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"testdata/from_json.dta",
		"reg Y X C1 C2 i.firm, cluster(firm)",
		"来自 JSON",
	} {
		if !strings.Contains(text, want) {
			t.Errorf(".do 缺少 %q\n实际:\n%s", want, text)
		}
	}
}

func TestRun_RequestFileNotFound(t *testing.T) {
	err := run([]string{
		"-request", `C:\no\such\req.json`,
		"-output-do", filepath.Join(t.TempDir(), "out.do"),
	})
	if err == nil {
		t.Fatal("request 文件不存在应报错")
	}
}

func TestRun_RequestInvalidJSON(t *testing.T) {
	reqPath := writeTemp(t, "bad.json", "{not valid json")
	err := run([]string{
		"-request", reqPath,
		"-output-do", filepath.Join(t.TempDir(), "out.do"),
	})
	if err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}

func TestRun_InvalidModel_TemplateMissing(t *testing.T) {
	// 覆盖 10.3 后 iv 有模板但 did 缺 did_treatment 应报错
	outDo := filepath.Join(t.TempDir(), "out.do")
	err := run([]string{
		"-data", "x.dta",
		"-dependent", "y",
		"-core", "x",
		"-model", "did",
		"-output-do", outDo,
	})
	if err == nil {
		t.Fatal("model=did 缺 did_treatment 应报错")
	}
	if !strings.Contains(err.Error(), "did_treatment 必填") {
		t.Errorf("错误信息应含'did_treatment 必填'，实际: %v", err)
	}
}
