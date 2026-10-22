package design

import (
	"fmt"
	"strings"
)

// Variable 一个变量的定义。
type Variable struct {
	Name    string `json:"name"`    // 变量名，如"企业创新绩效"
	Symbol  string `json:"symbol"`  // 符号，如"Innov"，可选
	Measure string `json:"measure"` // 测度方式，如"专利申请量加1取自然对数"
	Source  string `json:"source"`  // 数据来源，可选
}

// DesignInput 用户提供的结构化研究设计输入。
// 这是补上"研究设计 / 实证结果"章节缺口的输入。
type DesignInput struct {
	// 基本信息
	Topic      string `json:"topic"`      // 研究主题
	Hypothesis string `json:"hypothesis"` // 核心假设，可选

	// 数据与样本
	DataSource string `json:"data_source"` // 数据来源
	Sample     string `json:"sample"`      // 样本范围
	SampleSize string `json:"sample_size"` // 样本量描述，可选
	Period     string `json:"period"`      // 样本期，可选

	// 变量
	DependentVar Variable   `json:"dependent_var"`
	CoreVar      Variable   `json:"core_var"`
	ControlVars  []Variable `json:"control_vars"`
	Mediators    []Variable `json:"mediators"`  // 中介变量，可选
	Moderators   []Variable `json:"moderators"` // 调节变量，可选

	// 模型与识别
	Model          string   `json:"model"`          // 模型类型，如"双向固定效应"
	FixedEffects   []string `json:"fixed_effects"`  // 固定效应，如["个体","年份"]
	Clustering     string   `json:"clustering"`     // 聚类层级，如"企业层面"
	Identification []string `json:"identification"` // 识别策略，如["滞后一期","工具变量"]
	Robustness     []string `json:"robustness"`     // 稳健性检验，可选

	// 补充说明
	Notes string `json:"notes"` // 其他说明，可选

	// 实证结果（可选，用于生成"实证结果"节；为空时走占位兜底）
	Results *ResultsInput `json:"results,omitempty"`
}

// Validate 检查必填字段，返回人类可读的错误。
func (d DesignInput) Validate() error {
	var missing []string
	if strings.TrimSpace(d.DataSource) == "" {
		missing = append(missing, "data_source（数据来源）")
	}
	if strings.TrimSpace(d.Sample) == "" {
		missing = append(missing, "sample（样本范围）")
	}
	if strings.TrimSpace(d.DependentVar.Name) == "" {
		missing = append(missing, "dependent_var.name（被解释变量名）")
	}
	if strings.TrimSpace(d.DependentVar.Measure) == "" {
		missing = append(missing, "dependent_var.measure（被解释变量测度）")
	}
	if strings.TrimSpace(d.CoreVar.Name) == "" {
		missing = append(missing, "core_var.name（核心解释变量名）")
	}
	if strings.TrimSpace(d.CoreVar.Measure) == "" {
		missing = append(missing, "core_var.measure（核心解释变量测度）")
	}
	if strings.TrimSpace(d.Model) == "" {
		missing = append(missing, "model（模型类型）")
	}
	if len(missing) > 0 {
		return fmt.Errorf("design.json 缺少必填字段: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ToPromptContext 把结构化输入转成给 LLM 的文本上下文。
// 只输出非空字段，避免给模型塞无意义的空行。
func (d DesignInput) ToPromptContext() string {
	var sb strings.Builder

	sb.WriteString("【研究设计输入】\n")
	if d.Topic != "" {
		sb.WriteString(fmt.Sprintf("研究主题：%s\n", d.Topic))
	}
	if d.Hypothesis != "" {
		sb.WriteString(fmt.Sprintf("核心假设：%s\n", d.Hypothesis))
	}

	sb.WriteString("\n数据与样本：\n")
	if d.DataSource != "" {
		sb.WriteString(fmt.Sprintf("- 数据来源：%s\n", d.DataSource))
	}
	if d.Sample != "" {
		sb.WriteString(fmt.Sprintf("- 样本范围：%s\n", d.Sample))
	}
	if d.Period != "" {
		sb.WriteString(fmt.Sprintf("- 样本期：%s\n", d.Period))
	}
	if d.SampleSize != "" {
		sb.WriteString(fmt.Sprintf("- 样本量：%s\n", d.SampleSize))
	}

	sb.WriteString("\n变量定义：\n")
	sb.WriteString(fmt.Sprintf("- 被解释变量 %s：%s\n", d.DependentVar.Name, d.DependentVar.Measure))
	if d.DependentVar.Source != "" {
		sb.WriteString(fmt.Sprintf("  数据来源：%s\n", d.DependentVar.Source))
	}
	sb.WriteString(fmt.Sprintf("- 核心解释变量 %s：%s\n", d.CoreVar.Name, d.CoreVar.Measure))
	if d.CoreVar.Source != "" {
		sb.WriteString(fmt.Sprintf("  数据来源：%s\n", d.CoreVar.Source))
	}
	if len(d.ControlVars) > 0 {
		sb.WriteString("- 控制变量：\n")
		for _, v := range d.ControlVars {
			sb.WriteString(fmt.Sprintf("  · %s：%s\n", v.Name, v.Measure))
		}
	}
	if len(d.Mediators) > 0 {
		sb.WriteString("- 中介变量：\n")
		for _, v := range d.Mediators {
			sb.WriteString(fmt.Sprintf("  · %s：%s\n", v.Name, v.Measure))
		}
	}
	if len(d.Moderators) > 0 {
		sb.WriteString("- 调节变量：\n")
		for _, v := range d.Moderators {
			sb.WriteString(fmt.Sprintf("  · %s：%s\n", v.Name, v.Measure))
		}
	}

	sb.WriteString("\n模型与识别：\n")
	if d.Model != "" {
		sb.WriteString(fmt.Sprintf("- 模型类型：%s\n", d.Model))
	}
	if len(d.FixedEffects) > 0 {
		sb.WriteString(fmt.Sprintf("- 固定效应：%s\n", strings.Join(d.FixedEffects, "、")))
	}
	if d.Clustering != "" {
		sb.WriteString(fmt.Sprintf("- 聚类层级：%s\n", d.Clustering))
	}
	if len(d.Identification) > 0 {
		sb.WriteString(fmt.Sprintf("- 识别策略：%s\n", strings.Join(d.Identification, "、")))
	}
	if len(d.Robustness) > 0 {
		sb.WriteString(fmt.Sprintf("- 稳健性检验：%s\n", strings.Join(d.Robustness, "、")))
	}

	if d.Notes != "" {
		sb.WriteString(fmt.Sprintf("\n补充说明：%s\n", d.Notes))
	}

	return sb.String()
}

// RenderSection 用设计输入直接拼出“研究设计”章节正文，不走 LLM。
// 输出 100% 稳定，变量名、模型设定与 design.json 完全一致。
func (d DesignInput) RenderSection() string {
	var sb strings.Builder

	sb.WriteString("本研究以")
	if d.Topic != "" {
		sb.WriteString(d.Topic)
		sb.WriteString("为核心问题，")
	}
	sb.WriteString("基于")
	if d.DataSource != "" {
		sb.WriteString(d.DataSource)
	} else {
		sb.WriteString("公开数据库")
	}
	sb.WriteString("构建实证样本。")
	if d.Sample != "" {
		sb.WriteString("样本范围为")
		sb.WriteString(d.Sample)
		sb.WriteString("。")
	}
	if d.Period != "" {
		sb.WriteString("样本期为")
		sb.WriteString(d.Period)
		sb.WriteString("。")
	}
	if d.SampleSize != "" {
		sb.WriteString("最终得到")
		sb.WriteString(d.SampleSize)
		sb.WriteString("。")
	}
	sb.WriteString("\n\n")

	sb.WriteString("被解释变量为")
	sb.WriteString(d.DependentVar.Name)
	if d.DependentVar.Symbol != "" {
		sb.WriteString("（")
		sb.WriteString(d.DependentVar.Symbol)
		sb.WriteString("）")
	}
	sb.WriteString("，采用")
	sb.WriteString(d.DependentVar.Measure)
	sb.WriteString("进行测度")
	if d.DependentVar.Source != "" {
		sb.WriteString("，数据来源于")
		sb.WriteString(d.DependentVar.Source)
	}
	sb.WriteString("。")

	sb.WriteString("核心解释变量为")
	sb.WriteString(d.CoreVar.Name)
	if d.CoreVar.Symbol != "" {
		sb.WriteString("（")
		sb.WriteString(d.CoreVar.Symbol)
		sb.WriteString("）")
	}
	sb.WriteString("，采用")
	sb.WriteString(d.CoreVar.Measure)
	sb.WriteString("进行测度")
	if d.CoreVar.Source != "" {
		sb.WriteString("，数据来源于")
		sb.WriteString(d.CoreVar.Source)
	}
	sb.WriteString("。")

	if len(d.ControlVars) > 0 {
		sb.WriteString("控制变量包括：")
		var parts []string
		for _, v := range d.ControlVars {
			p := v.Name
			if v.Symbol != "" {
				p += "（" + v.Symbol + "）"
			}
			if v.Measure != "" {
				p += "，以" + v.Measure + "测度"
			}
			parts = append(parts, p)
		}
		sb.WriteString(strings.Join(parts, "；"))
		sb.WriteString("。")
	}

	if len(d.Mediators) > 0 {
		sb.WriteString("中介变量包括：")
		var parts []string
		for _, v := range d.Mediators {
			p := v.Name
			if v.Measure != "" {
				p += "（" + v.Measure + "）"
			}
			parts = append(parts, p)
		}
		sb.WriteString(strings.Join(parts, "；"))
		sb.WriteString("。")
	}

	if len(d.Moderators) > 0 {
		sb.WriteString("调节变量包括：")
		var parts []string
		for _, v := range d.Moderators {
			p := v.Name
			if v.Measure != "" {
				p += "（" + v.Measure + "）"
			}
			parts = append(parts, p)
		}
		sb.WriteString(strings.Join(parts, "；"))
		sb.WriteString("。")
	}
	sb.WriteString("\n\n")

	sb.WriteString("在模型设定上，本文采用")
	sb.WriteString(d.Model)
	sb.WriteString("进行估计")
	if len(d.FixedEffects) > 0 {
		sb.WriteString("，控制")
		sb.WriteString(strings.Join(d.FixedEffects, "与"))
		sb.WriteString("固定效应")
	}
	if d.Clustering != "" {
		sb.WriteString("，并使用")
		sb.WriteString(d.Clustering)
	}
	sb.WriteString("。")

	if len(d.Identification) > 0 {
		sb.WriteString("为缓解内生性问题，本文采用以下识别策略：")
		var parts []string
		for i, s := range d.Identification {
			parts = append(parts, fmt.Sprintf("（%d）%s", i+1, s))
		}
		sb.WriteString(strings.Join(parts, "；"))
		sb.WriteString("。")
	}

	if len(d.Robustness) > 0 {
		sb.WriteString("稳健性检验包括：")
		var parts []string
		for i, s := range d.Robustness {
			parts = append(parts, fmt.Sprintf("（%d）%s", i+1, s))
		}
		sb.WriteString(strings.Join(parts, "；"))
		sb.WriteString("。")
	}

	if d.Notes != "" {
		sb.WriteString("\n\n")
		sb.WriteString(d.Notes)
	}

	return sb.String()
}
