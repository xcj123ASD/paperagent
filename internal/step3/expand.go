package step3

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"paperagent/internal/design"
	"paperagent/internal/llm"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
)

// llmTimeout 单次 LLM 调用的超时上限。
// 一节写 1000–2000 字，正常 30–60 秒完成，90 秒是安全上限。
const llmTimeout = 90 * time.Second

// Stage 逐节扩写正文，填充 Section.Content
type Stage struct {
	LLM          llm.Client
	Mode         model.Mode
	Design       *design.DesignInput
	PolishDesign bool
}

func (Stage) Name() string { return "step3_litreview" }

// callLLM 带超时的 LLM 调用包装，防止单节卡死挂住全流程。
func (s Stage) callLLM(req llm.ExpandSectionRequest) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), llmTimeout)
	defer cancel()
	return s.LLM.ExpandSection(ctx, req)
}

// designContext 返回研究设计的 prompt 上下文，Design 为 nil 时返回空串。
func (s Stage) designContext() string {
	if s.Design == nil {
		return ""
	}
	return s.Design.ToPromptContext()
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
		sec := &f.Sections[i]

		if strings.TrimSpace(sec.Content) != "" {
			continue
		}

		// 研究设计节：有 Design 输入时优先模板；polish 开启时先试 LLM 润色
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
					continue
				}
				ctx.Logf("step3_expand",
					fmt.Sprintf("章节 %q LLM 润色失败或过短，回落模板", sec.Name))
			}
			sec.Content = s.Design.RenderSection()
			ctx.Logf("step3_expand",
				fmt.Sprintf("章节 %q 用模板生成，%d 字", sec.Name, len([]rune(sec.Content))))
			continue
		}

		// 无 LLM 时，无法生成的章节记 RiskHint 后继续
		if s.LLM == nil {
			f.RiskHints = append(f.RiskHints,
				fmt.Sprintf("章节 %q 无 LLM 可用，未生成", sec.Name))
			continue
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
		}

		content, err := s.callLLM(req)
		if err != nil {
			f.RiskHints = append(f.RiskHints,
				fmt.Sprintf("章节 %q 正文生成失败：%v", sec.Name, err))
			continue
		}
		content = strings.TrimSpace(content)

		// 空内容或过短：重试一次
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

		// 仍然过短：记 RiskHint，写占位符，避免空白页
		if len([]rune(content)) < 100 {
			f.RiskHints = append(f.RiskHints,
				fmt.Sprintf("章节 %q 生成内容过短（%d 字），需人工补充", sec.Name, len([]rune(content))))
			content = fmt.Sprintf("*（本节内容生成不足，可能因该章节需要真实文献或数据支撑。建议在此补充：%s）*",
				strings.Join(sec.KeyPoints, "；"))
		}

		sec.Content = content
		ctx.Logf("step3_expand",
			fmt.Sprintf("章节 %q 完成，%d 字（预算 %d）", sec.Name, len([]rune(sec.Content)), sec.WordBudget))
	}

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