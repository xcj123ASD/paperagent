package design

import (
	"strings"
	"testing"
)

// 场景1：results 为 nil（用户没填 results），五段都要出现，且基准回归要有占位提示
func TestRenderResultsSectionNil(t *testing.T) {
	out := RenderResultsSection(nil)
	wants := []string{
		"### 描述性统计",
		"### 基准回归结果",
		"### 机制检验",
		"### 异质性分析",
		"### 稳健性检验",
		"此处需填入系数",
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("nil 输出缺少 %q\n实际输出:\n%s", w, out)
		}
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("RenderResultsSection(nil) 返回空字符串，违反不空白原则")
	}
}

// 场景2：results 全填，五个字段都要出现在正文里
func TestRenderResultsSectionFull(t *testing.T) {
	out := RenderResultsSection(&ResultsInput{
		MainCoef:      "0.052",
		MainSE:        "0.015",
		Significance:  "1%",
		Direction:     "正向",
		Robustness:    []string{"替换被解释变量", "剔除疫情年份"},
		Mechanism:     []MechanismInput{{Name: "融资约束", Result: "中介效应显著"}},
		Heterogeneity: []HeteroInput{{Dimension: "企业规模", Result: "大企业效应更强"}},
	})
	wants := []string{
		"0.052",
		"0.015",
		"1%",
		"为正",
		"替换被解释变量",
		"剔除疫情年份",
		"融资约束",
		"企业规模",
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("full 输出缺少 %q\n实际输出:\n%s", w, out)
		}
	}
	if strings.Contains(out, "此处需填入系数") {
		t.Error("填了 MainCoef 后不应再出现占位提示")
	}
}

// 场景3：只填主系数，机制/异质性/稳健性为空 → 主系数出现，其余走占位，不 panic
func TestRenderResultsSectionPartial(t *testing.T) {
	out := RenderResultsSection(&ResultsInput{MainCoef: "0.052"})
	if !strings.Contains(out, "0.052") {
		t.Error("MainCoef 未出现在输出")
	}
	if !strings.Contains(out, "此处需补充机制检验") {
		t.Error("机制为空时应出占位")
	}
	if !strings.Contains(out, "此处需补充异质性分析") {
		t.Error("异质性为空时应出占位")
	}
	if !strings.Contains(out, "此处需补充稳健性检验") {
		t.Error("稳健性为空时应出占位")
	}
}
