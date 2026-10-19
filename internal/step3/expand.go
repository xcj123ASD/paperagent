package step3

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"paperagent/internal/llm"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
)

// Stage 逐节扩写正文，填充 Section.Content
type Stage struct{ LLM llm.Client }

func (Stage) Name() string { return "step3_litreview" }

func (s Stage) Run(ctx *pipeline.Context, input []byte) ([]byte, error) {
	ctx.Logf("step3_expand", fmt.Sprintf("收到输入 %d 字节", len(input)))

	var f model.Frame
	if err := json.Unmarshal(input, &f); err != nil {
		return nil, fmt.Errorf("step3_expand: 解析 Frame 失败: %w", err)
	}
	if len(f.Sections) == 0 {
		return nil, fmt.Errorf("step3_expand: sections 为空")
	}

	if s.LLM == nil {
		ctx.Logf("step3_expand", "LLM 为 nil，跳过正文生成")
		return json.MarshalIndent(f, "", "  ")
	}

	articleType := f.ArticleType
	if articleType == "" {
		articleType = "实证型"
	}

	for i := range f.Sections {
		sec := &f.Sections[i]

		if strings.TrimSpace(sec.Content) != "" {
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
		}

		content, err := s.LLM.ExpandSection(context.Background(), req)
		if err != nil {
			f.RiskHints = append(f.RiskHints,
				fmt.Sprintf("章节 %q 正文生成失败：%v", sec.Name, err))
			continue
		}
		sec.Content = strings.TrimSpace(content)
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