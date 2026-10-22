package design

import (
	"fmt"
	"strings"
)

// ResultsInput 是 design.json 中可选的 "results" 段。
// 全部字段可空；为空时 RenderResultsSection 输出可执行占位，不留空白。
type ResultsInput struct {
	MainCoef     string   `json:"main_coef,omitempty"`
	MainSE       string   `json:"main_se,omitempty"`
	Significance string   `json:"significance,omitempty"`
	Direction    string   `json:"direction,omitempty"`
	Robustness   []string `json:"robustness,omitempty"`

	Descriptive   string           `json:"descriptive,omitempty"`
	Mechanism     []MechanismInput `json:"mechanism,omitempty"`
	Heterogeneity []HeteroInput    `json:"heterogeneity,omitempty"`
}

type MechanismInput struct {
	Name   string `json:"name,omitempty"`
	Result string `json:"result,omitempty"`
}

type HeteroInput struct {
	Dimension string `json:"dimension,omitempty"`
	Result    string `json:"result,omitempty"`
}

// RenderResultsSection 生成"实证结果"节正文。
func RenderResultsSection(r *ResultsInput) string {
	var b strings.Builder

	b.WriteString("### 描述性统计\n\n")
	if r != nil && strings.TrimSpace(r.Descriptive) != "" {
		b.WriteString("表1报告主要变量的描述性统计结果。" + strings.TrimSpace(r.Descriptive) + "\n\n")
	} else {
		b.WriteString("表1报告主要变量的描述性统计结果（此处需补充样本量、均值、标准差、最小值与最大值）。\n\n")
	}

	b.WriteString("### 基准回归结果\n\n")
	b.WriteString(renderBaseline(r))
	b.WriteString("\n\n")

	b.WriteString("### 机制检验\n\n")
	b.WriteString(renderMechanism(r))
	b.WriteString("\n\n")

	b.WriteString("### 异质性分析\n\n")
	b.WriteString(renderHeterogeneity(r))
	b.WriteString("\n\n")

	b.WriteString("### 稳健性检验\n\n")
	b.WriteString(renderRobustness(r))

	return strings.TrimRight(b.String(), "\n") + "\n"
}

func renderBaseline(r *ResultsInput) string {
	if r == nil || strings.TrimSpace(r.MainCoef) == "" {
		return "表2报告基准回归结果，列(1)–(4)逐步加入控制变量与固定效应（此处需填入系数、标准误与显著性）。"
	}
	coef := strings.TrimSpace(r.MainCoef)
	se := strings.TrimSpace(r.MainSE)
	sig := strings.TrimSpace(r.Significance)
	dir := strings.TrimSpace(r.Direction)

	var b strings.Builder
	b.WriteString("表2报告基准回归结果，列(1)–(4)逐步加入控制变量与固定效应。")
	fmt.Fprintf(&b, "核心解释变量的估计系数为 %s", coef)
	if se != "" {
		fmt.Fprintf(&b, "（标准误 %s）", se)
	}
	if sig != "" {
		fmt.Fprintf(&b, "，在 %s 水平上显著", sig)
	}
	switch dir {
	case "正向", "正":
		b.WriteString("为正")
	case "负向", "负":
		b.WriteString("为负")
	case "":
	default:
		fmt.Fprintf(&b, "，方向为%s", dir)
	}
	b.WriteString("。")
	return b.String()
}

func renderMechanism(r *ResultsInput) string {
	if r == nil || len(r.Mechanism) == 0 {
		return "（此处需补充机制检验：中介变量、检验方法与结果。）"
	}
	var parts []string
	for i, m := range r.Mechanism {
		name := strings.TrimSpace(m.Name)
		res := strings.TrimSpace(m.Result)
		if name == "" && res == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("(%d) 以 %s 为中介变量，检验结果为：%s。", i+1, name, res))
	}
	if len(parts) == 0 {
		return "（此处需补充机制检验：中介变量、检验方法与结果。）"
	}
	return "本文进一步考察作用机制：" + strings.Join(parts, " ")
}

func renderHeterogeneity(r *ResultsInput) string {
	if r == nil || len(r.Heterogeneity) == 0 {
		return "（此处需补充异质性分析：分组维度与各组结论。）"
	}
	var parts []string
	for i, h := range r.Heterogeneity {
		dim := strings.TrimSpace(h.Dimension)
		res := strings.TrimSpace(h.Result)
		if dim == "" && res == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("(%d) 按 %s 分组，%s。", i+1, dim, res))
	}
	if len(parts) == 0 {
		return "（此处需补充异质性分析：分组维度与各组结论。）"
	}
	return "本文从以下维度进行异质性分析：" + strings.Join(parts, " ")
}

func renderRobustness(r *ResultsInput) string {
	if r == nil || len(r.Robustness) == 0 {
		return "（此处需补充稳健性检验：替换核心变量、调整样本区间、更换估计方法等。）"
	}
	var items []string
	for _, s := range r.Robustness {
		s = strings.TrimSpace(s)
		if s != "" {
			items = append(items, s)
		}
	}
	if len(items) == 0 {
		return "（此处需补充稳健性检验：替换核心变量、调整样本区间、更换估计方法等。）"
	}
	return "本文通过以下方式检验基准结果的稳健性：" + strings.Join(items, "、") + "，结果均保持稳健。"
}
