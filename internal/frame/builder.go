package frame

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"paperagent/internal/llm"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
)

type Builder struct {
	Rules CatalogRules
	Mode  model.Mode
	LLM   llm.Client
}

type CatalogRules map[string]struct {
	Columns map[string]ColumnRule
}

type ColumnRule struct {
	MinWords, MaxWords int
	EmpiricalRequired  bool
	DefaultStructure   []string
}

var roleMap = map[string]string{
	"引言":        "提出问题与研究意义",
	"文献综述":      "梳理既有研究并找缺口",
	"文献综述与研究假设": "提出研究假设",
	"研究设计":      "数据来源、变量与模型",
	"实证结果":      "主回归与稳健性检验",
	"分析与讨论":     "机制分析与讨论",
	"结论":        "总结与政策启示",
	"结论与政策启示":   "总结与建议",
}

// DefaultRules 兼容旧调用方，默认按 thesis 模式返回
func DefaultRules() CatalogRules {
	return DefaultRulesWithMode(model.ModeThesis)
}

// DefaultRulesWithMode 按论文模式返回栏目规则
func DefaultRulesWithMode(mode model.Mode) CatalogRules {
	base := CatalogRules{
		"中国行政管理": {Columns: map[string]ColumnRule{
			"公共管理前沿": {8000, 12000, false, []string{"引言", "文献综述与研究假设", "研究设计", "实证结果", "结论与政策启示"}},
		}},
	}
	if mode == model.ModeJournal {
		// TODO(M3): 期刊模式专属结构，需体现理论贡献、研究缺口、假设推导
		return base
	}
	return base
}

func fallbackColumnRule(totalWords int) ColumnRule {
	return ColumnRule{
		MinWords:         totalWords - 1000,
		MaxWords:         totalWords + 1000,
		DefaultStructure: []string{"引言", "文献综述", "研究设计", "分析与讨论", "结论"},
	}
}

// mode 零值统一当 thesis
func (b *Builder) mode() model.Mode {
	if b.Mode == "" {
		return model.ModeThesis
	}
	return b.Mode
}

// Build 兼容旧调用方，默认按"实证型"处理
func (b *Builder) Build(topic, targetJournal, column string, totalWords int) (model.Frame, error) {
	return b.BuildWithType(topic, targetJournal, column, totalWords, "实证型")
}

// BuildWithType 显式指定文章类型，供 Stage.Run 透传 step1 的 ArticleType
func (b *Builder) BuildWithType(topic, targetJournal, column string, totalWords int, articleType string) (model.Frame, error) {
	f := model.Frame{
		Title: topic, TargetJournal: targetJournal, Column: column, WordCount: totalWords,
		Mode:     b.mode(),
		TitleAlt: []string{}, Keywords: []string{}, RiskHints: []string{}, FigureTablePlan: []model.FigTable{},
	}

	if totalWords <= 0 {
		return f, fmt.Errorf("totalWords 必须为正数，当前为 %d", totalWords)
	}
	if articleType == "" {
		articleType = "实证型"
	}

	rules, ok := b.Rules[targetJournal]
	if !ok {
		f.RiskHints = append(f.RiskHints, fmt.Sprintf("期刊 %s 无栏目规则，已使用通用模板", targetJournal))
		fallback := CatalogRules{
			"_": {Columns: map[string]ColumnRule{
				"_": fallbackColumnRule(totalWords),
			}},
		}
		rules = fallback["_"]
		column = "_"
	} else if column == "" {
		f.RiskHints = append(f.RiskHints, "未指定栏目，已使用通用模板")
	}

	cr, ok := rules.Columns[column]
	if !ok {
		f.RiskHints = append(f.RiskHints, fmt.Sprintf("栏目 %q 无专属规则，已使用通用模板", column))
		cr = fallbackColumnRule(totalWords)
	}

	if cr.MinWords > 0 && totalWords < cr.MinWords {
		f.RiskHints = append(f.RiskHints, fmt.Sprintf("总字数 %d 低于栏目下限 %d", totalWords, cr.MinWords))
	}
	if cr.MaxWords > 0 && totalWords > cr.MaxWords {
		f.RiskHints = append(f.RiskHints, fmt.Sprintf("总字数 %d 高于栏目上限 %d", totalWords, cr.MaxWords))
	}

	structure := cr.DefaultStructure
	if len(structure) == 0 {
		structure = fallbackColumnRule(totalWords).DefaultStructure
	}
	n := len(structure)
	if totalWords < n {
		return f, fmt.Errorf("总字数 %d 小于章节数 %d，无法分配", totalWords, n)
	}
	base := totalWords / n
	for i, name := range structure {
		w := base
		if i == n-1 {
			w = totalWords - base*(n-1)
		}
		f.Sections = append(f.Sections, model.Section{
			Name: name, Role: roleOf(name), KeyPoints: []string{},
			RequiredEvidence: evidenceOf(name), WordBudget: w, MustCite: []string{}, Forbidden: forbiddenOf(name),
		})
	}

	if b.LLM != nil {
		for i := range f.Sections {
			s := &f.Sections[i]
			if len(s.KeyPoints) > 0 {
				continue
			}
			points, err := b.LLM.KeyPoints(context.Background(), llm.KeyPointRequest{
				Topic:       topic,
				Section:     s.Name,
				Role:        s.Role,
				ArticleType: articleType,
				MaxPoints:   4,
				Mode:        b.mode(),
			})
			if err != nil {
				f.RiskHints = append(f.RiskHints, fmt.Sprintf("章节 %q LLM 生成要点失败：%v", s.Name, err))
				continue
			}
			s.KeyPoints = points
		}
	}

	if err := ValidateL1(f, totalWords); err != nil {
		return f, err
	}
	return f, nil
}

func ValidateL1(f model.Frame, total int) error {
	if strings.TrimSpace(f.Title) == "" {
		return errors.New("title 为空")
	}
	if len(f.Sections) == 0 {
		return errors.New("sections 为空")
	}
	sum := 0
	hasLit, hasConcl := false, false
	for i, s := range f.Sections {
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("section[%d] name 为空", i)
		}
		if s.WordBudget <= 0 {
			return fmt.Errorf("section[%d] %q 字数预算非正：%d", i, s.Name, s.WordBudget)
		}
		sum += s.WordBudget
		if isLiteratureSection(s.Name) {
			hasLit = true
		}
		if isConclusionSection(s.Name) {
			hasConcl = true
		}
	}
	if sum != total {
		return fmt.Errorf("字数预算不闭合：各节之和=%d，总字数=%d", sum, total)
	}
	if !hasLit {
		return errors.New("缺少文献综述类章节")
	}
	if !hasConcl {
		return errors.New("缺少结论类章节")
	}
	return nil
}

func isLiteratureSection(name string) bool {
	return strings.Contains(name, "文献综述") || strings.Contains(name, "文献回顾")
}

func isConclusionSection(name string) bool {
	return strings.Contains(name, "结论") || strings.Contains(name, "结语")
}

func roleOf(name string) string {
	if v, ok := roleMap[name]; ok {
		return v
	}
	return "待补充"
}

func evidenceOf(name string) []string {
	if strings.Contains(name, "研究设计") {
		return []string{"数据来源说明", "变量定义表"}
	}
	if strings.Contains(name, "实证结果") {
		return []string{"回归结果表", "稳健性检验表"}
	}
	return []string{}
}

func forbiddenOf(name string) []string {
	if strings.Contains(name, "引言") {
		return []string{"回归结果", "公式推导"}
	}
	if strings.Contains(name, "结论") || strings.Contains(name, "结语") {
		return []string{"新的变量定义", "新的文献引用"}
	}
	return []string{}
}

// Stage 包装，供流水线调用
type Stage struct{ B *Builder }

func (Stage) Name() string { return "step2_frame" }

func (s Stage) Run(ctx *pipeline.Context, input []byte) ([]byte, error) {
	ctx.Logf("step2_frame", fmt.Sprintf("收到输入 %d 字节", len(input)))

	var s1 model.Step1Output
	if err := json.Unmarshal(input, &s1); err != nil {
		return nil, fmt.Errorf("step2_frame: 解析 step1 输入失败: %w", err)
	}
	if len(s1.Journals) == 0 {
		return nil, fmt.Errorf("step2_frame: step1 未返回任何期刊")
	}

	j := s1.Journals[0]
	column := ""
	if len(j.Columns) > 0 {
		column = j.Columns[0]
	}
	totalWords := resolveTotalWords(j.WordMin, j.WordMax)
	if totalWords <= 0 {
		totalWords = 10000
	}

	b := s.B
	if b == nil {
		b = &Builder{Rules: DefaultRules()}
	}

	f, err := b.BuildWithType(s1.Topic, j.Name, column, totalWords, s1.ArticleType)
	if err != nil {
		return nil, err
	}

	if j.EmpiricalOnly && s1.ArticleType != "实证型" {
		f.RiskHints = append(f.RiskHints, fmt.Sprintf("期刊《%s》仅收实证型，当前文章类型为 %s", j.Name, s1.ArticleType))
	}

	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}
	return out, nil
}

func resolveTotalWords(min, max int) int {
	if min > 0 && max > 0 {
		return (min + max) / 2
	}
	if max > 0 {
		return max
	}
	if min > 0 {
		return min
	}
	return 0
}
