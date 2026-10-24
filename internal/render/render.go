package render

import (
	"strings"

	"paperagent/internal/model"
)

// Markdown 把 Frame 和参考文献文本渲染成完整 Markdown。
func Markdown(f model.Frame, refsText string) string {
	var sb strings.Builder
	sb.WriteString("# " + f.Title + "\n\n")
	if len(f.Keywords) > 0 {
		sb.WriteString("**关键词：** " + strings.Join(f.Keywords, "；") + "\n\n")
	}
	if f.AbstractCN != "" {
		sb.WriteString("## 摘要\n\n" + f.AbstractCN + "\n\n")
	}
	for _, sec := range f.Sections {
		sb.WriteString("## " + sec.Name + "\n\n")
		content := strings.TrimSpace(sec.Content)
		content = strings.TrimPrefix(content, "## "+sec.Name)
		content = strings.TrimSpace(content)
		if content != "" {
			sb.WriteString(content + "\n\n")
		} else {
			sb.WriteString("*（本节未生成）*\n\n")
		}
	}
	if refsText != "" {
		sb.WriteString("## 参考文献\n\n")
		for _, line := range strings.Split(refsText, "\n") {
			if strings.TrimSpace(line) != "" {
				sb.WriteString(line + "\n\n")
			}
		}
	}
	if len(f.RiskHints) > 0 {
		sb.WriteString("---\n\n**风险提示**\n\n")
		for _, h := range f.RiskHints {
			sb.WriteString("- " + h + "\n")
		}
	}
	return sb.String()
}
