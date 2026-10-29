package step3

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"paperagent/internal/cite"
	"paperagent/internal/design"
	"paperagent/internal/litreview"
	"paperagent/internal/llm"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
)

// llmTimeout 单次 LLM 调用的超时上限。
const llmTimeout = 90 * time.Second

// Stage 逐节扩写正文，填充 Section.Content
type Stage struct {
	LLM          llm.Client
	Mode         model.Mode
	Design       *design.DesignInput
	PolishDesign bool
	Refs         []cite.Reference
	ProgressPath string // A5：进度文件路径，空则禁用断点续跑
}

// progressSection 进度文件中单节记录。
type progressSection struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// progressData 进度文件结构。
type progressData struct {
	Sections []progressSection `json:"sections"`
}

func (Stage) Name() string { return "step3_litreview" }

// callLLM 带超时的 LLM 调用包装。
func (s Stage) callLLM(req llm.ExpandSectionRequest) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), llmTimeout)
	defer cancel()
	return s.LLM.ExpandSection(ctx, req)
}

func (s Stage) designContext() string {
	if s.Design == nil {
		return ""
	}
	return s.Design.ToPromptContext()
}

// fallbackContent 在 LLM 失败或内容过短时，按章节类型选模板兜底。
// 返回 (内容, 是否命中模板)。未命中返回 ("", false)，由调用方写占位符。
func (s Stage) fallbackContent(sec *model.Section) (string, bool) {
	switch {
	case s.Design != nil && isIntroSection(sec.Name):
		return s.Design.RenderIntro(), true
	case s.Design != nil && isConclusionSection(sec.Name):
		return s.Design.RenderConclusion(), true
	}
	return "", false
}

// loadProgress 从进度文件按 section name 合并已完成内容到 f。
// 文件不存在、解析失败、ProgressPath 为空时静默跳过。
func (s Stage) loadProgress(f *model.Frame) {
	if s.ProgressPath == "" {
		return
	}
	data, err := os.ReadFile(s.ProgressPath)
	if err != nil {
		return
	}
	var pd progressData
	if err := json.Unmarshal(data, &pd); err != nil {
		return
	}
	byName := make(map[string]string, len(pd.Sections))
	for _, ps := range pd.Sections {
		byName[ps.Name] = ps.Content
	}
	for i := range f.Sections {
		if c, ok := byName[f.Sections[i].Name]; ok && strings.TrimSpace(c) != "" {
			f.Sections[i].Content = c
		}
	}
}

// saveProgress 把当前所有非空 section 写入进度文件。
// 写失败静默忽略——进度文件是辅助，不中断主流程。
func (s Stage) saveProgress(f *model.Frame) {
	if s.ProgressPath == "" {
		return
	}
	var pd progressData
	for _, sec := range f.Sections {
		if strings.TrimSpace(sec.Content) != "" {
			pd.Sections = append(pd.Sections, progressSection{Name: sec.Name, Content: sec.Content})
		}
	}
	data, err := json.MarshalIndent(pd, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.ProgressPath, data, 0644)
}

// removeProgress 删除进度文件。
func (s Stage) removeProgress() {
	if s.ProgressPath == "" {
		return
	}
	_ = os.Remove(s.ProgressPath)
}

// processSection 处理单节。原 Run 循环体提取，continue 改 return。
func (s Stage) processSection(ctx *pipeline.Context, f *model.Frame, i int, articleType string, mode model.Mode) {
	sec := &f.Sections[i]

	if s.Design != nil && strings.Contains(sec.Name, "研究设计") {
		if s.PolishDesign && s.LLM != nil {
			req := llm.ExpandSectionRequest{
				Topic:            f.Title,
				Section:          sec.Name,
				Role:             sec.Role,
				KeyPoints:        sec.KeyPoints,
				WordBudget:       sec.WordBudget,
				ArticleType:      articleType,
				PreviousSummary:  buildPreviousSummary(f.Sections, i),
				RequiredEvidence: sec.RequiredEvidence,
				Forbidden:        sec.Forbidden,
				Mode:             mode,
				DesignContext:    s.designContext(),
			}
			polished, err := s.callLLM(req)
			polished = strings.TrimSpace(polished)
			if err == nil && len([]rune(polished)) >= 200 {
				sec.Content = polished
				ctx.Logf("step3_expand",
					fmt.Sprintf("章节 %q 用 LLM 润色，%d 字", sec.Name, len([]rune(polished))))
				return
			}
			ctx.Logf("step3_expand",
				fmt.Sprintf("章节 %q LLM 润色失败或过短，回落模板", sec.Name))
		}
		sec.Content = s.Design.RenderSection()
		ctx.Logf("step3_expand",
			fmt.Sprintf("章节 %q 用模板生成，%d 字", sec.Name, len([]rune(sec.Content))))
		return
	}

	// 实证结果节走模板（与"研究设计"节同模式），优先于 LLM
	if s.Design != nil && isResultsSection(sec.Name) {
		sec.Content = design.RenderResultsSection(s.Design.Results)
		ctx.Logf("step3_expand",
			fmt.Sprintf("章节 %q 用模板生成，%d 字", sec.Name, len([]rune(sec.Content))))
		return
	}

	// 文献综述节：骨架兜底 + LLM 带 refs 润色，失败回落骨架
	if isLitReviewSection(sec.Name) {
		skeleton := litreview.RenderSkeleton(s.Refs)
		if s.LLM != nil {
			req := llm.ExpandSectionRequest{
				Topic:            f.Title,
				Section:          sec.Name,
				Role:             sec.Role,
				KeyPoints:        sec.KeyPoints,
				WordBudget:       sec.WordBudget,
				ArticleType:      articleType,
				PreviousSummary:  buildPreviousSummary(f.Sections, i),
				RequiredEvidence: sec.RequiredEvidence,
				Forbidden:        sec.Forbidden,
				Mode:             mode,
				DesignContext:    s.designContext(),
				References:       formatRefsForPrompt(s.Refs),
			}
			polished, err := s.callLLM(req)
			polished = strings.TrimSpace(polished)
			if err == nil && len([]rune(polished)) >= 200 {
				// 超预算 1.25 倍截断
				if sec.WordBudget > 0 && len([]rune(polished)) > sec.WordBudget*5/4 {
					origLen := len([]rune(polished))
					limit := sec.WordBudget * 5 / 4
					polished = truncateAtSentence(polished, limit)
					f.RiskHints = append(f.RiskHints,
						fmt.Sprintf("章节 %q 超预算（原 %d 字 → 截断至 %d 字，丢弃 %d 字），需人工补",
							sec.Name, origLen, len([]rune(polished)), origLen-len([]rune(polished))))
				}
				sec.Content = polished
				ctx.Logf("step3_expand",
					fmt.Sprintf("章节 %q 用 LLM 带文献润色，%d 字", sec.Name, len([]rune(polished))))
				return
			}
			f.RiskHints = append(f.RiskHints,
				fmt.Sprintf("章节 %q LLM 润色失败或过短，已回落骨架，需人工改写", sec.Name))
			ctx.Logf("step3_expand",
				fmt.Sprintf("章节 %q LLM 润色失败或过短，回落骨架", sec.Name))
		}
		sec.Content = skeleton
		ctx.Logf("step3_expand",
			fmt.Sprintf("章节 %q 用骨架生成，%d 字", sec.Name, len([]rune(sec.Content))))
		return
	}

	if s.LLM == nil {
		f.RiskHints = append(f.RiskHints,
			fmt.Sprintf("章节 %q 无 LLM 可用，未生成", sec.Name))
		return
	}

	req := llm.ExpandSectionRequest{
		Topic:            f.Title,
		Section:          sec.Name,
		Role:             sec.Role,
		KeyPoints:        sec.KeyPoints,
		WordBudget:       sec.WordBudget,
		ArticleType:      articleType,
		PreviousSummary:  buildPreviousSummary(f.Sections, i),
		RequiredEvidence: sec.RequiredEvidence,
		Forbidden:        sec.Forbidden,
		Mode:             mode,
		DesignContext:    s.designContext(),
		References:       formatRefsForPrompt(s.Refs),
	}

	content, err := s.callLLM(req)
	if err != nil {
		f.RiskHints = append(f.RiskHints,
			fmt.Sprintf("章节 %q 正文生成失败：%v", sec.Name, err))
		// 失败时也尝试模板兜底，避免引言/结论空白
		if fb, ok := s.fallbackContent(sec); ok {
			sec.Content = fb
			ctx.Logf("step3_expand",
				fmt.Sprintf("章节 %q LLM 失败，用模板兜底，%d 字", sec.Name, len([]rune(fb))))
		}
		return
	}
	content = strings.TrimSpace(content)

	if len([]rune(content)) < 100 {
		ctx.Logf("step3_expand", fmt.Sprintf("章节 %q 返回内容过短（%d 字），重试一次", sec.Name, len([]rune(content))))
		retryContent, retryErr := s.callLLM(req)
		if retryErr == nil {
			retryContent = strings.TrimSpace(retryContent)
			if len([]rune(retryContent)) >= 100 {
				content = retryContent
			}
		}
	}

	// 仍然过短：引言/结论走模板兜底，其他节记 RiskHint 写占位符
	if len([]rune(content)) < 100 {
		if fb, ok := s.fallbackContent(sec); ok {
			origLen := len([]rune(content))
			content = fb
			f.RiskHints = append(f.RiskHints,
				fmt.Sprintf("章节 %q LLM 生成内容过短（%d 字），已用模板兜底，需人工改写", sec.Name, origLen))
			ctx.Logf("step3_expand",
				fmt.Sprintf("章节 %q 内容过短，用模板兜底，%d 字", sec.Name, len([]rune(content))))
		} else {
			f.RiskHints = append(f.RiskHints,
				fmt.Sprintf("章节 %q 生成内容过短（%d 字，预算 %d 字），需人工补充",
					sec.Name, len([]rune(content)), sec.WordBudget))
			content = fmt.Sprintf("*（本节内容生成不足，可能因该章节需要真实文献或数据支撑。建议在此补充：%s）*",
				strings.Join(sec.KeyPoints, "；"))
		}
	} else if sec.WordBudget > 0 {
		// 内容有效：校验实际字数与预算的偏差
		actualLen := len([]rune(content))
		ratio := float64(actualLen) / float64(sec.WordBudget)
		if ratio < 0.7 || ratio > 1.3 {
			f.RiskHints = append(f.RiskHints,
				fmt.Sprintf("章节 %q 字数偏差较大：实际 %d 字，预算 %d 字（偏差 %+.0f%%）",
					sec.Name, actualLen, sec.WordBudget, (ratio-1)*100))
		}
	}

	// 超预算 1.25 倍截断，防止 LLM 无视字数约束
	if sec.WordBudget > 0 && len([]rune(content)) > sec.WordBudget*5/4 {
		origLen := len([]rune(content))
		limit := sec.WordBudget * 5 / 4
		content = truncateAtSentence(content, limit)
		f.RiskHints = append(f.RiskHints,
			fmt.Sprintf("章节 %q 超预算（原 %d 字 → 截断至 %d 字，丢弃 %d 字），需人工补",
				sec.Name, origLen, len([]rune(content)), origLen-len([]rune(content))))
	}

	sec.Content = content
	ctx.Logf("step3_expand",
		fmt.Sprintf("章节 %q 完成，%d 字（预算 %d）", sec.Name, len([]rune(sec.Content)), sec.WordBudget))
}

func (s Stage) Run(ctx *pipeline.Context, input []byte) ([]byte, error) {
	ctx.Logf("step3_expand", fmt.Sprintf("收到输入 %d 字节", len(input)))

	var f model.Frame
	if err := json.Unmarshal(input, &f); err != nil {
		return nil, fmt.Errorf("step3_expand: 解析 Frame 失败: %w", err)
	}
	if len(f.Sections) == 0 {
		return nil, fmt.Errorf("step3_expand: sections 为空")
	}

	// A5：加载进度文件，按 section name 合并已完成内容
	s.loadProgress(&f)

	if s.LLM == nil && s.Design == nil {
		ctx.Logf("step3_expand", "LLM 为 nil 且无 Design，跳过正文生成")
		return json.MarshalIndent(f, "", "  ")
	}

	articleType := f.ArticleType
	if articleType == "" {
		articleType = "实证型"
	}

	mode := s.Mode
	if mode == "" {
		mode = f.Mode
	}

	for i := range f.Sections {
		if strings.TrimSpace(f.Sections[i].Content) != "" {
			continue
		}
		s.processSection(ctx, &f, i, articleType, mode)
		s.saveProgress(&f)
	}

	// A5：全部完成，删除进度文件
	s.removeProgress()

	return json.MarshalIndent(f, "", "  ")
}

func buildPreviousSummary(sections []model.Section, currentIdx int) string {
	if currentIdx == 0 {
		return ""
	}
	start := currentIdx - 3
	if start < 0 {
		start = 0
	}
	var sb strings.Builder
	sb.WriteString("已写章节：\n")
	for i := start; i < currentIdx; i++ {
		sec := sections[i]
		sb.WriteString(fmt.Sprintf("- %s（%s）", sec.Name, sec.Role))
		if len(sec.KeyPoints) > 0 {
			sb.WriteString("：")
			sb.WriteString(strings.Join(sec.KeyPoints, "；"))
		}
		sb.WriteString("\n")
		if i == currentIdx-1 && sec.Content != "" {
			tail := []rune(sec.Content)
			if len(tail) > 200 {
				tail = tail[len(tail)-200:]
			}
			sb.WriteString("上一节结尾：…")
			sb.WriteString(string(tail))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// isResultsSection 判断章节名是否为"实证结果"类章节。
// 与"研究设计"节同模式走模板，避免 LLM 拒写导致空白。
func isResultsSection(name string) bool {
	for _, kw := range []string{"实证结果", "实证分析", "回归结果"} {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}

// isLitReviewSection 判断章节名是否为"文献综述"类章节。
// 走骨架 + LLM 带 refs 润色模式，骨架兜底不空白。
func isLitReviewSection(name string) bool {
	for _, kw := range []string{"文献综述", "文献回顾", "文献评述"} {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}

// isIntroSection 判断章节名是否为"引言"类章节。
// LLM 失败或过短时用引言模板兜底。
func isIntroSection(name string) bool {
	for _, kw := range []string{"引言", "绪论", "导论"} {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}

// isConclusionSection 判断章节名是否为"结论"类章节。
// LLM 失败或过短时用结论模板兜底。
func isConclusionSection(name string) bool {
	for _, kw := range []string{"结论", "结语"} {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}

// formatRefsForPrompt 把参考文献转成给 LLM 的简短列表，形如 "张三(2023)《标题》"。
func formatRefsForPrompt(refs []cite.Reference) []string {
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		author := "佚名"
		if len(r.Authors) > 0 {
			author = r.Authors[0]
			if len(r.Authors) > 1 {
				author += "等"
			}
		}
		year := "n.d."
		if r.Year > 0 {
			year = fmt.Sprintf("%d", r.Year)
		}
		title := r.Title
		if title == "" {
			title = "（无标题）"
		}
		out = append(out, fmt.Sprintf("%s(%s)《%s》", author, year, title))
	}
	return out
}

// truncateAtSentence 在 limit 字内找最后一个句末标点截断，避免砍断句子。
func truncateAtSentence(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	for i := limit - 1; i >= 0; i-- {
		switch runes[i] {
		case '。', '！', '？', '.', '!', '?':
			return string(runes[:i+1])
		}
	}
	return string(runes[:limit])
}
