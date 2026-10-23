package design

import (
	"fmt"
	"strings"
)

// RenderIntro 用设计输入拼出"引言"节正文，不走 LLM。
// 无 Design 数据时也保证不空白，输出可执行占位。
func (d DesignInput) RenderIntro() string {
	var sb strings.Builder

	sb.WriteString("本研究围绕")
	if strings.TrimSpace(d.Topic) != "" {
		sb.WriteString("“")
		sb.WriteString(strings.TrimSpace(d.Topic))
		sb.WriteString("”")
	} else {
		sb.WriteString("（此处需补充研究主题）")
	}
	sb.WriteString("展开。")

	if strings.TrimSpace(d.DataSource) != "" {
		sb.WriteString("在制造业转型升级与数字经济加速渗透的背景下，基于")
		sb.WriteString(strings.TrimSpace(d.DataSource))
		sb.WriteString("构建企业层面样本")
		if strings.TrimSpace(d.Sample) != "" {
			sb.WriteString("，样本范围为")
			sb.WriteString(strings.TrimSpace(d.Sample))
		}
		sb.WriteString("，")
	} else {
		sb.WriteString("在数字经济与产业升级的背景下，")
	}
	sb.WriteString("本文试图回答：核心解释变量如何影响被解释变量，其作用机制与边界条件为何。\n\n")

	if strings.TrimSpace(d.Hypothesis) != "" {
		sb.WriteString("核心假设为：")
		sb.WriteString(strings.TrimSpace(d.Hypothesis))
		sb.WriteString("。\n\n")
	} else {
		sb.WriteString("核心假设为：（此处需补充研究假设）。\n\n")
	}

	sb.WriteString("本文的研究意义体现在三个方面：")
	sb.WriteString("理论上，将核心机制纳入统一分析框架，回应现有研究在机制黑箱与边界条件方面的不足；")
	sb.WriteString("方法上，结合")
	if strings.TrimSpace(d.Model) != "" {
		sb.WriteString(strings.TrimSpace(d.Model))
	} else {
		sb.WriteString("计量模型")
	}
	sb.WriteString("与内生性处理策略，为净效应识别提供更稳健的证据；")
	sb.WriteString("实践上，为相关主体优化决策与政策设计提供参考。\n\n")

	sb.WriteString("本文余下部分安排如下：第二部分为文献综述与研究假设；第三部分为研究设计；第四部分为实证结果；第五部分为结论与政策启示。")

	return sb.String()
}

// RenderConclusion 用设计输入拼出"结论"节正文，不走 LLM。
// 优先使用 d.Results 的真实系数与方向；为空时出占位，不空白。
func (d DesignInput) RenderConclusion() string {
	var sb strings.Builder

	sb.WriteString("本文围绕")
	if strings.TrimSpace(d.Topic) != "" {
		sb.WriteString("“")
		sb.WriteString(strings.TrimSpace(d.Topic))
		sb.WriteString("”")
	} else {
		sb.WriteString("（此处需补充研究主题）")
	}
	sb.WriteString("展开实证检验。")

	// 主要发现
	sb.WriteString("主要发现如下：")
	if d.Results != nil && strings.TrimSpace(d.Results.MainCoef) != "" {
		r := d.Results
		sb.WriteString("核心解释变量的估计系数为 ")
		sb.WriteString(strings.TrimSpace(r.MainCoef))
		if strings.TrimSpace(r.MainSE) != "" {
			sb.WriteString("（标准误 ")
			sb.WriteString(strings.TrimSpace(r.MainSE))
			sb.WriteString("）")
		}
		if strings.TrimSpace(r.Significance) != "" {
			sb.WriteString("，在 ")
			sb.WriteString(strings.TrimSpace(r.Significance))
			sb.WriteString(" 水平上显著")
		}
		switch strings.TrimSpace(r.Direction) {
		case "正向", "正":
			sb.WriteString("为正")
		case "负向", "负":
			sb.WriteString("为负")
		}
		sb.WriteString("，表明核心解释变量对被解释变量具有")
		if strings.TrimSpace(r.Direction) != "" {
			sb.WriteString(strings.TrimSpace(r.Direction))
		} else {
			sb.WriteString("一定")
		}
		sb.WriteString("影响。")
	} else {
		sb.WriteString("（此处需补充基准回归的核心系数、显著性水平与方向。）")
	}

	if d.Results != nil && len(d.Results.Robustness) > 0 {
		sb.WriteString("该结论在")
		var items []string
		for _, r := range d.Results.Robustness {
			if s := strings.TrimSpace(r); s != "" {
				items = append(items, s)
			}
		}
		if len(items) > 0 {
			sb.WriteString(strings.Join(items, "、"))
			sb.WriteString("等稳健性检验后依然成立。")
		} else {
			sb.WriteString("（此处需补充稳健性检验结论。）")
		}
	} else {
		sb.WriteString("（此处需补充稳健性检验结论。）")
	}
	sb.WriteString("\n\n")

	// 理论贡献
	sb.WriteString("从理论上看，本文的贡献在于：")
	sb.WriteString("将信息处理与知识整合等机制纳入统一分析框架，丰富了相关领域的机制研究；")
	sb.WriteString("结合")
	if strings.TrimSpace(d.Model) != "" {
		sb.WriteString(strings.TrimSpace(d.Model))
	} else {
		sb.WriteString("计量模型")
	}
	sb.WriteString("与多种内生性处理策略，为净效应识别提供了更稳健的微观证据。\n\n")

	// 政策启示
	sb.WriteString("从实践上看，本文的结论具有以下启示：")
	sb.WriteString("（1）企业应结合自身禀赋，分阶段推进相关工作，避免一次性重投入；")
	sb.WriteString("（2）应重视组织能力与人力资本的配套建设，使投入转化为可持续的能力提升；")
	sb.WriteString("（3）政府应完善基础设施并实施差异化支持政策，缓解异质性与能力差距。\n\n")

	// 局限
	sb.WriteString("本文仍存在一定局限：")
	sb.WriteString("（1）变量测度受数据可得性约束，可能存在测量误差；")
	sb.WriteString("（2）机制检验受中介变量选择影响，未来可拓展更多渠道；")
	sb.WriteString("（3）样本范围与时期有限，结论的外推需谨慎。")
	if strings.TrimSpace(d.Notes) != "" {
		sb.WriteString("\n\n")
		sb.WriteString(strings.TrimSpace(d.Notes))
	}

	return sb.String()
}

// Ensure fmt is used (placeholder for future formatted output)
var _ = fmt.Sprintf
