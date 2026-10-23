package design

import (
	"strings"
	"testing"
)

func TestRenderIntroEmptyDesign(t *testing.T) {
	var d DesignInput
	out := d.RenderIntro()
	if strings.TrimSpace(out) == "" {
		t.Fatal("空 Design 的引言不应为空")
	}
	for _, w := range []string{"（此处需补充研究主题）", "（此处需补充研究假设）", "本文余下部分安排如下"} {
		if !strings.Contains(out, w) {
			t.Errorf("空 Design 引言缺少 %q\n实际：%s", w, out)
		}
	}
}

func TestRenderIntroWithDesign(t *testing.T) {
	d := DesignInput{
		Topic:      "数字化转型对制造业企业创新绩效的影响",
		DataSource: "CSMAR 数据库",
		Sample:     "2010-2023 年 A 股制造业上市公司",
		Hypothesis: "数字化转型促进制造业企业创新绩效",
		Model:      "双向固定效应模型",
	}
	out := d.RenderIntro()
	for _, w := range []string{"数字化转型对制造业企业创新绩效的影响", "CSMAR 数据库", "2010-2023 年 A 股制造业上市公司", "数字化转型促进制造业企业创新绩效", "双向固定效应模型"} {
		if !strings.Contains(out, w) {
			t.Errorf("引言缺少 %q\n实际：%s", w, out)
		}
	}
}

func TestRenderConclusionNoResults(t *testing.T) {
	d := DesignInput{
		Topic: "测试主题",
		Model: "双向固定效应模型",
	}
	out := d.RenderConclusion()
	if strings.TrimSpace(out) == "" {
		t.Fatal("结论不应为空")
	}
	for _, w := range []string{"测试主题", "（此处需补充基准回归的核心系数", "理论上看", "实践上看", "本文仍存在一定局限"} {
		if !strings.Contains(out, w) {
			t.Errorf("无 Results 结论缺少 %q\n实际：%s", w, out)
		}
	}
}

func TestRenderConclusionWithResults(t *testing.T) {
	d := DesignInput{
		Topic: "测试主题",
		Model: "双向固定效应模型",
		Results: &ResultsInput{
			MainCoef:     "0.052",
			MainSE:       "0.015",
			Significance: "1%",
			Direction:    "正向",
			Robustness:   []string{"替换被解释变量", "剔除疫情年份"},
		},
	}
	out := d.RenderConclusion()
	for _, w := range []string{"0.052", "0.015", "1%", "为正", "替换被解释变量", "剔除疫情年份"} {
		if !strings.Contains(out, w) {
			t.Errorf("有 Results 结论缺少 %q\n实际：%s", w, out)
		}
	}
	if strings.Contains(out, "（此处需补充基准回归的核心系数") {
		t.Error("填了 MainCoef 后不应再出现基准系数占位")
	}
}

func TestRenderConclusionNegativeDirection(t *testing.T) {
	d := DesignInput{
		Results: &ResultsInput{
			MainCoef:  "-0.02",
			Direction: "负向",
		},
	}
	out := d.RenderConclusion()
	if !strings.Contains(out, "为负") {
		t.Errorf("负向系数应输出'为负'，实际：%s", out)
	}
}
