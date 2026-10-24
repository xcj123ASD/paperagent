package litreview

import (
	"strings"
	"testing"

	"paperagent/internal/cite"
)

// refLines 只返回输出里以 "- " 开头的文献条目行，排除占位文案。
// 避免断言撞上 "（此处需补充...发现...）" 这类模板固定文案。
func refLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- ") {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestRenderSkeleton_L3Full(t *testing.T) {
	refs := []cite.Reference{
		{
			Authors:    []string{"赵宸宇", "王文春"},
			Title:      "数字化转型如何影响企业全要素生产率",
			Year:       2021,
			Method:     "固定效应模型",
			Conclusion: "数字化转型显著提升企业全要素生产率",
			Relation:   "支持",
		},
	}
	lines := refLines(RenderSkeleton(refs))
	if len(lines) != 1 {
		t.Fatalf("应只有 1 条文献行，实际 %d", len(lines))
	}
	line := lines[0]
	for _, want := range []string{
		"赵宸宇等(2021)",
		"采用固定效应模型",
		"研究了《数字化转型如何影响企业全要素生产率》",
		"发现数字化转型显著提升企业全要素生产率",
		"（与本文关系：支持）",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("L3 文献行缺少 %q\n文献行: %s", want, line)
		}
	}
}

func TestRenderSkeleton_AbstractOnly(t *testing.T) {
	refs := []cite.Reference{
		{
			Authors:  []string{"张三"},
			Title:    "测试标题",
			Year:     2022,
			Abstract: "本文研究了 X 对 Y 的影响",
		},
	}
	lines := refLines(RenderSkeleton(refs))
	if len(lines) != 1 {
		t.Fatalf("应只有 1 条文献行，实际 %d", len(lines))
	}
	line := lines[0]
	if !strings.Contains(line, "摘要指出本文研究了 X 对 Y 的影响") {
		t.Errorf("只有摘要时应输出'摘要指出'，实际: %s", line)
	}
	if strings.Contains(line, "发现") {
		t.Errorf("无 Conclusion 时文献行不应输出'发现'，实际: %s", line)
	}
}

func TestRenderSkeleton_MetadataOnly(t *testing.T) {
	refs := []cite.Reference{
		{Authors: []string{"李四"}, Title: "元数据文献", Year: 2020},
	}
	lines := refLines(RenderSkeleton(refs))
	if len(lines) != 1 {
		t.Fatalf("应只有 1 条文献行，实际 %d", len(lines))
	}
	line := lines[0]
	if !strings.Contains(line, "李四(2020)研究了《元数据文献》") {
		t.Errorf("纯元数据应输出基础句，实际: %s", line)
	}
	for _, forbidden := range []string{"采用", "发现", "摘要指出"} {
		if strings.Contains(line, forbidden) {
			t.Errorf("纯元数据文献行不应含 %q，实际: %s", forbidden, line)
		}
	}
}

func TestRenderSkeleton_Mixed(t *testing.T) {
	// 同一组里混合：一篇 L3，一篇纯元数据
	refs := []cite.Reference{
		{Authors: []string{"A"}, Title: "L3 文献", Year: 2021, Conclusion: "结论A"},
		{Authors: []string{"B"}, Title: "元数据文献", Year: 2021},
	}
	lines := refLines(RenderSkeleton(refs))
	if len(lines) != 2 {
		t.Fatalf("应有 2 条文献行，实际 %d", len(lines))
	}
	if !strings.Contains(lines[0], "发现结论A") {
		t.Errorf("L3 文献行应输出'发现'，实际: %s", lines[0])
	}
	if !strings.Contains(lines[1], "B(2021)研究了《元数据文献》。") {
		t.Errorf("元数据文献行应回落基础句，实际: %s", lines[1])
	}
}
