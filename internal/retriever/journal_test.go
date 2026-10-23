package retriever

import (
	"strings"
	"testing"

	"paperagent/internal/model"
)

// weights1 是四个维度等权的便捷权重，和为 1。
var weights1 = DisciplineWeights{Discipline: 0.25, Column: 0.25, Quality: 0.25, Hotness: 0.25}

// twoJournals 构造两个差异明显的期刊：
// A 全匹配（数字治理/CSSCI/数字栏目），B 全不匹配（社会学/其他/论文栏目）。
func twoJournals() []model.Journal {
	return []model.Journal{
		{
			Name: "期刊A", Discipline: "公共管理", Sub: "数字治理", Level: "CSSCI",
			Columns: []string{"数字治理"},
		},
		{
			Name: "期刊B", Discipline: "社会学", Sub: "社会分层", Level: "其他",
			Columns: []string{"论文"},
		},
	}
}

// ---- MatchJournals ----

func TestMatchJournals_EmptyCatalog(t *testing.T) {
	out := MatchJournals("t", "d", "s", []string{"k"}, nil, weights1)
	if out != nil {
		t.Errorf("空 catalog 应返回 nil，实际: %+v", out)
	}
	out = MatchJournals("t", "d", "s", []string{"k"}, []model.Journal{}, weights1)
	if out != nil {
		t.Errorf("空切片应返回 nil，实际: %+v", out)
	}
}

func TestMatchJournals_SingleItemNormalizedToHalf(t *testing.T) {
	// 只有一个期刊时，每个维度 b<=a，归一化返回 0.5，加权和应为 0.5
	js := []model.Journal{{Name: "唯一", Level: "CSSCI"}}
	out := MatchJournals("t", "d", "s", []string{"k"}, js, weights1)
	if len(out) != 1 {
		t.Fatalf("应返回 1 个期刊，实际 %d", len(out))
	}
	if out[0].Score != 0.5 {
		t.Errorf("单元素归一化应得 0.5，实际 %v", out[0].Score)
	}
}

func TestMatchJournals_SortsByScore(t *testing.T) {
	out := MatchJournals("数字化转型", "公共管理", "数字治理",
		[]string{"数字", "治理"}, twoJournals(), weights1)
	if len(out) != 2 {
		t.Fatalf("应返回 2 个，实际 %d", len(out))
	}
	if out[0].Name != "期刊A" {
		t.Errorf("全匹配期刊A应排第一，实际第一是 %q（Score=%v）", out[0].Name, out[0].Score)
	}
	if out[0].Score < out[1].Score {
		t.Errorf("排序错误：A=%v 应 >= B=%v", out[0].Score, out[1].Score)
	}
}

func TestMatchJournals_ReturnsAllItems(t *testing.T) {
	// 不截断：返回数量应等于输入数量
	js := twoJournals()
	out := MatchJournals("t", "d", "s", []string{"k"}, js, weights1)
	if len(out) != len(js) {
		t.Errorf("应返回全部 %d 个，实际 %d", len(js), len(out))
	}
}

func TestMatchJournals_ScoreInRange(t *testing.T) {
	js := twoJournals()
	out := MatchJournals("t", "公共管理", "数字治理", []string{"数字"}, js, weights1)
	for _, j := range out {
		if j.Score < 0 || j.Score > 1 {
			t.Errorf("Score 应在 [0,1] 内，实际 %v（%s）", j.Score, j.Name)
		}
	}
}

func TestMatchJournals_ZeroWeights(t *testing.T) {
	// 权重全 0 不 panic，所有 Score 应为 0
	js := twoJournals()
	out := MatchJournals("t", "d", "s", []string{"k"}, js, DisciplineWeights{})
	if len(out) != len(js) {
		t.Fatalf("应返回全部，实际 %d", len(out))
	}
	for _, j := range out {
		if j.Score != 0 {
			t.Errorf("权重全 0 时 Score 应为 0，实际 %v（%s）", j.Score, j.Name)
		}
	}
}

func TestMatchJournals_BreakdownFormat(t *testing.T) {
	out := MatchJournals("t", "d", "s", []string{"k"}, twoJournals(), weights1)
	for _, j := range out {
		for _, want := range []string{"学科=", "栏目=", "质量=", "时效="} {
			if !strings.Contains(j.Breakdown, want) {
				t.Errorf("Breakdown 应含 %q，实际 %q", want, j.Breakdown)
			}
		}
	}
}

// ---- discScore ----

func TestDiscScore_SubContained(t *testing.T) {
	j := model.Journal{Discipline: "公共管理", Sub: "数字治理"}
	got := discScore(j, "公共管理", "数字治理", nil)
	if got != 1.0 {
		t.Errorf("子学科完全匹配应 1.0，实际 %v", got)
	}
}

func TestDiscScore_SubPartialMatch(t *testing.T) {
	// sub 互相包含（"数字治理" 与 "数字治理研究"）
	j := model.Journal{Discipline: "公共管理", Sub: "数字治理"}
	got := discScore(j, "公共管理", "数字治理研究", nil)
	if got != 1.0 {
		t.Errorf("子学科互相包含应 1.0，实际 %v", got)
	}
}

func TestDiscScore_DisciplineOnly(t *testing.T) {
	j := model.Journal{Discipline: "公共管理", Sub: "数字治理"}
	got := discScore(j, "公共管理", "完全不同", nil)
	if got != 0.8 {
		t.Errorf("仅学科匹配应 0.8，实际 %v", got)
	}
}

func TestDiscScore_KeywordMatch(t *testing.T) {
	j := model.Journal{Discipline: "公共管理", Sub: "数字治理"}
	kw := map[string]bool{"数字": true}
	got := discScore(j, "不同学科", "不同子学科", kw)
	if got != 0.5 {
		t.Errorf("关键词命中应 0.5，实际 %v", got)
	}
}

func TestDiscScore_NoMatch(t *testing.T) {
	j := model.Journal{Discipline: "公共管理", Sub: "数字治理"}
	kw := map[string]bool{"量子": true}
	got := discScore(j, "不同学科", "不同子学科", kw)
	if got != 0.1 {
		t.Errorf("无匹配应 0.1，实际 %v", got)
	}
}

// ---- columnScore ----

func TestColumnScore_NoKeywords(t *testing.T) {
	j := model.Journal{Columns: []string{"数字治理"}}
	got := columnScore(j, map[string]bool{})
	if got != 0.5 {
		t.Errorf("无关键词应 0.5，实际 %v", got)
	}
}

func TestColumnScore_PartialHit(t *testing.T) {
	j := model.Journal{Columns: []string{"数字治理", "公共管理"}}
	kw := map[string]bool{"数字": true, "量子": true}
	// "数字治理" 含"数字"命中，另一个 kw 不命中 → 1/2 = 0.5
	got := columnScore(j, kw)
	if got != 0.5 {
		t.Errorf("命中 1/2 应 0.5，实际 %v", got)
	}
}

func TestColumnScore_AllHit(t *testing.T) {
	j := model.Journal{Columns: []string{"数字治理"}}
	kw := map[string]bool{"数字": true, "治理": true}
	// "数字治理" 同时含"数字"和"治理"，但内层循环 break 只算一次
	// 所以 hit=1，len(kw)=2 → 0.5
	got := columnScore(j, kw)
	if got != 0.5 {
		t.Errorf("单栏目命中两关键词（break 只算一次）应 0.5，实际 %v", got)
	}
}

// ---- qualityScore ----

func TestQualityScore_AllLevels(t *testing.T) {
	cases := []struct {
		level string
		want  float64
	}{
		{"CSSCI", 1.0},
		{"CSSCI扩展", 0.85},
		{"北大核心", 0.7},
		{"其他", 0.4},
		{"SSCI", 0.4}, // 未知归"其他"
	}
	for _, c := range cases {
		t.Run(c.level, func(t *testing.T) {
			if got := qualityScore(c.level); got != c.want {
				t.Errorf("qualityScore(%q) = %v，期望 %v", c.level, got, c.want)
			}
		})
	}
}

// ---- hotnessScore ----

func TestHotnessScore_NoKeywords(t *testing.T) {
	got := hotnessScore(map[string]bool{})
	if got != 0.5 {
		t.Errorf("无关键词应 0.5，实际 %v", got)
	}
}

func TestHotnessScore_PartialHit(t *testing.T) {
	// "数字" 命中热门词，"量子" 不命中 → 1/2 = 0.5
	kw := map[string]bool{"数字": true, "量子": true}
	got := hotnessScore(kw)
	if got != 0.5 {
		t.Errorf("命中 1/2 应 0.5，实际 %v", got)
	}
}

func TestHotnessScore_AllHit(t *testing.T) {
	kw := map[string]bool{"数字": true, "治理": true, "智能": true}
	got := hotnessScore(kw)
	if got != 1.0 {
		t.Errorf("全部命中应 1.0，实际 %v", got)
	}
}

// ---- round / lowerSet ----

func TestRound(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0.123, 0.12},
		{0.125, 0.13}, // math.Round 四舍五入
		{0.999, 1.0},
		{1.0, 1.0},
		{0.0, 0.0},
	}
	for _, c := range cases {
		if got := round(c.in); got != c.want {
			t.Errorf("round(%v) = %v，期望 %v", c.in, got, c.want)
		}
	}
}

func TestLowerSet_TrimsWhitespace(t *testing.T) {
	got := lowerSet([]string{" 数字 ", "治理", ""})
	if !got["数字"] {
		t.Error("应 trim 后存入'数字'")
	}
	if !got["治理"] {
		t.Error("应存入'治理'")
	}
	if !got[""] {
		t.Error("空字符串也应存入（函数不跳过空串）")
	}
}
