package design

import (
"os"
"path/filepath"
"testing"
)

func writeTemp(t *testing.T, content string) string {
t.Helper()
dir := t.TempDir()
p := filepath.Join(dir, "design.json")
if err := os.WriteFile(p, []byte(content), 0644); err != nil {
t.Fatalf("写入临时文件失败: %v", err)
}
return p
}

func TestLoad_Valid(t *testing.T) {
js := `{
"topic": "测试主题",
"data_source": "CSMAR",
"sample": "A 股制造业",
"dependent_var": {"name": "创新", "measure": "专利数"},
"core_var": {"name": "数字化", "measure": "词频"},
"model": "双向固定效应"
}`
p := writeTemp(t, js)
d, err := Load(p)
if err != nil {
t.Fatalf("不应报错: %v", err)
}
if d.DataSource != "CSMAR" {
t.Errorf("DataSource 不符: %s", d.DataSource)
}
if d.DependentVar.Name != "创新" {
t.Errorf("DependentVar.Name 不符: %s", d.DependentVar.Name)
}
}

func TestLoad_MissingRequired(t *testing.T) {
js := `{"topic": "缺字段"}`
p := writeTemp(t, js)
if _, err := Load(p); err == nil {
t.Fatal("缺必填字段应报错")
}
}

func TestLoad_FileNotFound(t *testing.T) {
if _, err := Load("E:/nonexistent/design.json"); err == nil {
t.Fatal("文件不存在应报错")
}
}

func TestLoad_InvalidJSON(t *testing.T) {
p := writeTemp(t, "not json")
if _, err := Load(p); err == nil {
t.Fatal("非法 JSON 应报错")
}
}

func TestToPromptContext_ContainsKeyFields(t *testing.T) {
d := DesignInput{
Topic:      "测试主题",
DataSource: "CSMAR",
Sample:     "A 股制造业",
DependentVar: Variable{Name: "创新", Measure: "专利数"},
CoreVar:      Variable{Name: "数字化", Measure: "词频"},
Model:        "双向固定效应",
}
out := d.ToPromptContext()
for _, want := range []string{"测试主题", "CSMAR", "A 股制造业", "创新", "数字化", "双向固定效应"} {
if !contains(out, want) {
t.Errorf("ToPromptContext 缺少 %q", want)
}
}
}

func contains(s, sub string) bool {
return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
for i := 0; i+len(sub) <= len(s); i++ {
if s[i:i+len(sub)] == sub {
return i
}
}
return -1
}