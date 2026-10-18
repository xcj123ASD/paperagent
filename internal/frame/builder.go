package frame

import (
	"encoding/json"
	"errors"
	"fmt"

	"paperagent/internal/model"
	"paperagent/internal/pipeline"
)

var ErrNotImplemented = errors.New("LLM 路径尚未接入")

type Builder struct{ Rules CatalogRules }

type CatalogRules map[string]struct {
	Columns map[string]ColumnRule
}

type ColumnRule struct {
	MinWords, MaxWords    int
	EmpiricalRequired     bool
	DefaultStructure      []string
}

func DefaultRules() CatalogRules {
	return CatalogRules{
		"中国行政管理": {Columns: map[string]ColumnRule{
			"公共管理前沿": {8000, 12000, false, []string{"引言", "文献综述与研究假设", "研究设计", "实证结果", "结论与政策启示"}},
		}},
	}
}

func (b *Builder) Build(topic, targetJournal, column string, totalWords int) (model.Frame, error) {
	f := model.Frame{
		Title: topic, TargetJournal: targetJournal, Column: column, WordCount: totalWords,
		TitleAlt: []string{}, Keywords: []string{}, RiskHints: []string{}, FigureTablePlan: []model.FigTable{},
	}
	
	rules, ok := b.Rules[targetJournal]
	if !ok {
		f.RiskHints = append(f.RiskHints, fmt.Sprintf("期刊 %s 无栏目规则，已使用通用模板", targetJournal))
		fallback := CatalogRules{
			"_": {Columns: map[string]ColumnRule{
				"_": {totalWords - 1000, totalWords + 1000, false, []string{"引言", "文献综述", "研究设计", "分析与讨论", "结论"}},
			}},
		}
		rules = fallback["_"]
		column = "_"
	}

	cr, ok := rules.Columns[column]
	if !ok {
		cr = ColumnRule{totalWords - 1000, totalWords + 1000, false, []string{"引言", "文献综述", "研究设计", "结论"}}
	}

	n := len(cr.DefaultStructure)
	base := totalWords / n
	for i, name := range cr.DefaultStructure {
		w := base
		if i == n-1 {
			w = totalWords - base*(n-1)
		}
		f.Sections = append(f.Sections, model.Section{
			Name: name, Role: roleOf(name), KeyPoints: []string{},
			RequiredEvidence: evidenceOf(name), WordBudget: w, MustCite: []string{}, Forbidden: forbiddenOf(name),
		})
	}
	
	if err := ValidateL1(f, totalWords); err != nil {
		return f, err
	}
	return f, nil
}

func ValidateL1(f model.Frame, total int) error {
	if f.Title == "" {
		return errors.New("title 为空")
	}
	if len(f.Sections) == 0 {
		return errors.New("sections 为空")
	}
	sum := 0
	hasLit, hasConcl := false, false
	for i, s := range f.Sections {
		if s.Name == "" {
			return fmt.Errorf("section[%d] name 为空", i)
		}
		sum += s.WordBudget
		if s.Name == "文献综述" || s.Name == "文献综述与研究假设" {
			hasLit = true
		}
		if s.Name == "结论" || s.Name == "结论与政策启示" {
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

func roleOf(name string) string {
	m := map[string]string{
		"引言": "提出问题与研究意义", 
		"文献综述": "梳理既有研究并找缺口",
		"研究设计": "数据来源、变量与模型", 
		"实证结果": "主回归与稳健性检验",
		"结论": "总结与政策启示", 
		"文献综述与研究假设": "提出研究假设",
		"分析与讨论": "机制分析与讨论", 
		"结论与政策启示": "总结与建议",
	}
	if v, ok := m[name]; ok {
		return v
	}
	return "待补充"
}

func evidenceOf(name string) []string {
	if name == "研究设计" {
		return []string{"数据来源说明", "变量定义表"}
	}
	if name == "实证结果" {
		return []string{"回归结果表", "稳健性检验表"}
	}
	return []string{}
}

func forbiddenOf(name string) []string {
	if name == "引言" {
		return []string{"回归结果", "公式推导"}
	}
	if name == "结论" {
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
		return nil, err
	}
	if len(s1.Journals) == 0 {
		return nil, fmt.Errorf("step2_frame: step1 未返回任何期刊")
	}

	j := s1.Journals[0]
	column := ""
	if len(j.Columns) > 0 {
		column = j.Columns[0]
	}
	totalWords := (j.WordMin + j.WordMax) / 2
	if totalWords <= 0 {
		totalWords = 10000
	}

	f, err := s.B.Build(s1.Topic, j.Name, column, totalWords)
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
