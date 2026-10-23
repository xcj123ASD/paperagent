package render

import (
	"strings"
	"testing"

	"paperagent/internal/model"
)

func TestMarkdown_TitleOnly(t *testing.T) {
	f := model.Frame{Title: "测试标题"}
	out := Markdown(f, "")
	if !strings.HasPrefix(out, "# 测试标题\n\n") {
		t.Errorf("应以 # 标题开头，实际:\n%s", out)
	}
	if strings.Contains(out, "## 摘要") {
		t.Errorf("无摘要时不应有摘要节")
	}
	if strings.Contains(out, "## 参考文献") {
		t.Errorf("空 refsText 不应有参考文献节")
	}
	if strings.Contains(out, "**风险提示**") {
		t.Errorf("无 RiskHints 不应有风险提示节")
	}
}

func TestMarkdown_WithKeywords(t *testing.T) {
	f := model.Frame{Title: "T", Keywords: []string{"数字化", "创新", "治理"}}
	out := Markdown(f, "")
	if !strings.Contains(out, "**关键词：** 数字化；创新；治理") {
		t.Errorf("关键词应以'；'分隔，实际:\n%s", out)
	}
}

func TestMarkdown_EmptyKeywords(t *testing.T) {
	f := model.Frame{Title: "T", Keywords: []string{}}
	out := Markdown(f, "")
	if strings.Contains(out, "**关键词") {
		t.Errorf("空 Keywords 不应渲染关键词行，实际:\n%s", out)
	}
}

func TestMarkdown_WithAbstract(t *testing.T) {
	f := model.Frame{Title: "T", AbstractCN: "这是中文摘要。"}
	out := Markdown(f, "")
	if !strings.Contains(out, "## 摘要\n\n这是中文摘要。") {
		t.Errorf("摘要应正确渲染，实际:\n%s", out)
	}
}

func TestMarkdown_SectionWithContent(t *testing.T) {
	f := model.Frame{
		Title: "T",
		Sections: []model.Section{
			{Name: "引言", Content: "引言正文内容。"},
		},
	}
	out := Markdown(f, "")
	if !strings.Contains(out, "## 引言\n\n引言正文内容。") {
		t.Errorf("有内容的章节应正常渲染，实际:\n%s", out)
	}
	if strings.Contains(out, "本节未生成") {
		t.Errorf("有内容不应出占位符")
	}
}

func TestMarkdown_SectionWithoutContent(t *testing.T) {
	f := model.Frame{
		Title: "T",
		Sections: []model.Section{
			{Name: "实证结果", Content: ""},
		},
	}
	out := Markdown(f, "")
	if !strings.Contains(out, "## 实证结果\n\n*（本节未生成）*") {
		t.Errorf("空内容章节应出占位符，实际:\n%s", out)
	}
}

func TestMarkdown_MultipleSections(t *testing.T) {
	f := model.Frame{
		Title: "T",
		Sections: []model.Section{
			{Name: "引言", Content: "A"},
			{Name: "文献综述", Content: "B"},
			{Name: "研究设计", Content: ""},
		},
	}
	out := Markdown(f, "")
	for _, want := range []string{"## 引言\n\nA", "## 文献综述\n\nB", "## 研究设计\n\n*（本节未生成）*"} {
		if !strings.Contains(out, want) {
			t.Errorf("缺少 %q，实际:\n%s", want, out)
		}
	}
}

func TestMarkdown_WithRefs(t *testing.T) {
	refsText := "[1] 张三. 测试文献[J]. 测试期刊, 2023.\n[2] 李四. 另一篇[J]. 另一期刊, 2022."
	out := Markdown(model.Frame{Title: "T"}, refsText)
	if !strings.Contains(out, "## 参考文献") {
		t.Errorf("有 refsText 应有参考文献节，实际:\n%s", out)
	}
	if !strings.Contains(out, "[1] 张三") || !strings.Contains(out, "[2] 李四") {
		t.Errorf("两条文献应都出现，实际:\n%s", out)
	}
}

func TestMarkdown_RefsEmptyLinesFiltered(t *testing.T) {
	// refsText 含空行，应被跳过
	refsText := "[1] 文献A\n\n\n[2] 文献B\n"
	out := Markdown(model.Frame{Title: "T"}, refsText)
	// 不应出现连续三个以上换行（空行被过滤，每行之间只有 \n\n）
	if strings.Contains(out, "\n\n\n\n") {
		t.Errorf("空行应被过滤，实际含 4 个以上连续换行:\n%s", out)
	}
	if !strings.Contains(out, "[1] 文献A") || !strings.Contains(out, "[2] 文献B") {
		t.Errorf("非空行应保留，实际:\n%s", out)
	}
}

func TestMarkdown_WithRiskHints(t *testing.T) {
	f := model.Frame{
		Title:     "T",
		RiskHints: []string{"章节 X 字数偏少", "章节 Y 需人工补充"},
	}
	out := Markdown(f, "")
	if !strings.Contains(out, "**风险提示**") {
		t.Errorf("有 RiskHints 应有风险提示节，实际:\n%s", out)
	}
	if !strings.Contains(out, "- 章节 X 字数偏少") || !strings.Contains(out, "- 章节 Y 需人工补充") {
		t.Errorf("两条风险提示应都以 - 开头，实际:\n%s", out)
	}
}

func TestMarkdown_Full(t *testing.T) {
	f := model.Frame{
		Title:      "完整测试论文",
		Keywords:   []string{"K1", "K2"},
		AbstractCN: "摘要正文。",
		Sections: []model.Section{
			{Name: "引言", Content: "引言正文。"},
			{Name: "结论", Content: ""},
		},
		RiskHints: []string{"结论需补充"},
	}
	refsText := "[1] 张三. 测试文献. 2023."
	out := Markdown(f, refsText)

	// 顺序检查：标题 → 关键词 → 摘要 → 章节 → 参考文献 → 风险提示
	order := []string{
		"# 完整测试论文",
		"**关键词：** K1；K2",
		"## 摘要",
		"摘要正文。",
		"## 引言",
		"引言正文。",
		"## 结论",
		"*（本节未生成）*",
		"## 参考文献",
		"[1] 张三",
		"**风险提示**",
		"- 结论需补充",
	}
	pos := 0
	for _, want := range order {
		idx := strings.Index(out[pos:], want)
		if idx < 0 {
			t.Fatalf("缺少或顺序错误：%q 未出现在 %q 之后\n实际:\n%s", want, order[:len(order)-1], out)
		}
		pos += idx + len(want)
	}
}
