package frame

import (
	"encoding/json"
	"testing"

	"paperagent/internal/llm"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
)

func TestBuild_DefaultRulesHit(t *testing.T) {
	b := &Builder{Rules: DefaultRules()}
	f, err := b.Build("基层治理数字化", "中国行政管理", "公共管理前沿", 10000)
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	if len(f.Sections) != 5 {
		t.Fatalf("期望 5 节，实际 %d", len(f.Sections))
	}
	sum := 0
	for _, s := range f.Sections {
		sum += s.WordBudget
		if s.Role == "" || s.Role == "待补充" {
			t.Errorf("章节 %q 的 Role 未填充", s.Name)
		}
	}
	if sum != 10000 {
		t.Fatalf("字数预算不闭合：sum=%d", sum)
	}
	if len(f.RiskHints) != 0 {
		t.Fatalf("命中规则不应产生 RiskHints，实际: %v", f.RiskHints)
	}
}

func TestBuild_UnknownJournalFallback(t *testing.T) {
	b := &Builder{Rules: DefaultRules()}
	f, err := b.Build("t", "不存在的期刊", "", 8000)
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	if len(f.Sections) == 0 {
		t.Fatal("fallback 应生成章节")
	}
	if len(f.RiskHints) == 0 {
		t.Fatal("未知期刊应产生 RiskHints")
	}
	sum := 0
	for _, s := range f.Sections {
		sum += s.WordBudget
	}
	if sum != 8000 {
		t.Fatalf("fallback 字数不闭合：sum=%d", sum)
	}
}

func TestBuild_UnknownColumnFallback(t *testing.T) {
	b := &Builder{Rules: DefaultRules()}
	f, err := b.Build("t", "中国行政管理", "不存在的栏目", 8000)
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	if len(f.RiskHints) == 0 {
		t.Fatal("未知栏目应产生 RiskHints")
	}
}

func TestBuild_NonPositiveTotalWords(t *testing.T) {
	b := &Builder{Rules: DefaultRules()}
	if _, err := b.Build("t", "中国行政管理", "公共管理前沿", 0); err == nil {
		t.Fatal("totalWords=0 应报错")
	}
	if _, err := b.Build("t", "中国行政管理", "公共管理前沿", -100); err == nil {
		t.Fatal("totalWords<0 应报错")
	}
}

func TestBuild_TotalWordsSmallerThanSections(t *testing.T) {
	b := &Builder{Rules: DefaultRules()}
	// 5 节结构，总字数 3 无法分配
	if _, err := b.Build("t", "中国行政管理", "公共管理前沿", 3); err == nil {
		t.Fatal("总字数小于章节数应报错")
	}
}

func TestBuild_WordCountOutOfRangeHint(t *testing.T) {
	b := &Builder{Rules: DefaultRules()}
	// 规则区间 8000~12000
	low, err := b.Build("t", "中国行政管理", "公共管理前沿", 7000)
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	if !hasHintContaining(low.RiskHints, "低于栏目下限") {
		t.Fatalf("低于下限应产生提示，实际: %v", low.RiskHints)
	}

	high, err := b.Build("t", "中国行政管理", "公共管理前沿", 13000)
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	if !hasHintContaining(high.RiskHints, "高于栏目上限") {
		t.Fatalf("高于上限应产生提示，实际: %v", high.RiskHints)
	}
}

func TestValidateL1_ContainsMatch(t *testing.T) {
	f := model.Frame{
		Title: "x",
		Sections: []model.Section{
			{Name: "文献综述与研究假设", WordBudget: 500},
			{Name: "结论与政策启示", WordBudget: 500},
		},
	}
	if err := ValidateL1(f, 1000); err != nil {
		t.Fatalf("组合名应能命中：%v", err)
	}
}

func TestValidateL1_MissingLiterature(t *testing.T) {
	f := model.Frame{
		Title: "x",
		Sections: []model.Section{
			{Name: "引言", WordBudget: 500},
			{Name: "结论", WordBudget: 500},
		},
	}
	if err := ValidateL1(f, 1000); err == nil {
		t.Fatal("缺少文献综述类章节应报错")
	}
}

func TestValidateL1_MissingConclusion(t *testing.T) {
	f := model.Frame{
		Title: "x",
		Sections: []model.Section{
			{Name: "引言", WordBudget: 500},
			{Name: "文献综述", WordBudget: 500},
		},
	}
	if err := ValidateL1(f, 1000); err == nil {
		t.Fatal("缺少结论类章节应报错")
	}
}

func TestValidateL1_SumMismatch(t *testing.T) {
	f := model.Frame{
		Title: "x",
		Sections: []model.Section{
			{Name: "引言", WordBudget: 500},
			{Name: "文献综述", WordBudget: 500},
			{Name: "结论", WordBudget: 500},
		},
	}
	if err := ValidateL1(f, 1200); err == nil {
		t.Fatal("字数不闭合应报错")
	}
}

func TestValidateL1_EmptySectionName(t *testing.T) {
	f := model.Frame{
		Title: "x",
		Sections: []model.Section{
			{Name: "", WordBudget: 500},
			{Name: "文献综述", WordBudget: 500},
			{Name: "结论", WordBudget: 500},
		},
	}
	if err := ValidateL1(f, 1500); err == nil {
		t.Fatal("空章节名应报错")
	}
}

func TestRoleOf(t *testing.T) {
	cases := map[string]string{
		"引言":        "提出问题与研究意义",
		"文献综述与研究假设": "提出研究假设",
		"结论与政策启示":   "总结与建议",
		"未知章节":      "待补充",
	}
	for name, want := range cases {
		if got := roleOf(name); got != want {
			t.Errorf("roleOf(%q)=%q, want %q", name, got, want)
		}
	}
}

func TestEvidenceOf_Contains(t *testing.T) {
	got := evidenceOf("研究设计与数据来源")
	if len(got) == 0 {
		t.Fatal("包含“研究设计”的章节名应命中")
	}
}

func TestForbiddenOf_Contains(t *testing.T) {
	got := forbiddenOf("结论与政策启示")
	if len(got) == 0 {
		t.Fatal("包含“结论”的章节名应命中")
	}
}

func hasHintContaining(hints []string, sub string) bool {
	for _, h := range hints {
		if len(h) >= len(sub) && contains(h, sub) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestStageRun_OK(t *testing.T) {
	input := []byte(`{
		"topic": "基层治理数字化",
		"article_type": "实证型",
		"journals": [{
			"name": "中国行政管理",
			"columns": ["公共管理前沿"],
			"word_min": 9000,
			"word_max": 11000,
			"empirical_only": true
		}]
	}`)
	ctx := pipeline.NewContext(true, false)
	out, err := (Stage{}).Run(ctx, input)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	var f model.Frame
	if err := json.Unmarshal(out, &f); err != nil {
		t.Fatalf("输出不是合法 Frame JSON: %v", err)
	}
	sum := 0
	for _, s := range f.Sections {
		sum += s.WordBudget
	}
	if sum != f.WordCount {
		t.Fatalf("字数不闭合: sum=%d wordCount=%d", sum, f.WordCount)
	}
}

func TestStageRun_EmptyJournals(t *testing.T) {
	input := []byte(`{"topic":"t","article_type":"实证型","journals":[]}`)
	ctx := pipeline.NewContext(true, false)
	if _, err := (Stage{}).Run(ctx, input); err == nil {
		t.Fatal("空 journals 应报错")
	}
}

func TestBuild_LLMFillsKeyPoints(t *testing.T) {
	b := &Builder{Rules: DefaultRules(), LLM: llm.MockClient{}}
	f, err := b.Build("基层治理数字化", "中国行政管理", "公共管理前沿", 10000)
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	for _, s := range f.Sections {
		if len(s.KeyPoints) == 0 {
			t.Errorf("章节 %q 的 KeyPoints 未填充", s.Name)
		}
	}
}

func TestBuild_NilLLMKeepsKeyPointsEmpty(t *testing.T) {
	// LLM 为 nil 时不应报错，KeyPoints 保持为空，行为跟旧版一致
	b := &Builder{Rules: DefaultRules()}
	f, err := b.Build("基层治理数字化", "中国行政管理", "公共管理前沿", 10000)
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	for _, s := range f.Sections {
		if len(s.KeyPoints) != 0 {
			t.Errorf("LLM=nil 时章节 %q 不应有 KeyPoints，实际: %v", s.Name, s.KeyPoints)
		}
	}
}