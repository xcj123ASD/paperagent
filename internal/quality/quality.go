package quality

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"paperagent/internal/cite"
)

// Report 质量检测报告。
type Report struct {
	LitReviewOK       bool
	LitReviewIssues   []string
	CitationOK        bool
	CitationIssues    []string // 裸引用幻觉，判 FAIL
	CitationWarnings  []string // 有占位符提示的幻觉，判 WARN
	PlaceholderCount  int
	PlaceholderIssues []string
}

// AllPass 是否全部通过。占位符不阻塞——项目定位是"给草稿不给终稿"。
func (r Report) AllPass() bool {
	return r.LitReviewOK && r.CitationOK
}

// Check 跑全部检测。
func Check(md string, refs []cite.Reference) Report {
	litOK, litIssues := CheckLitReview(md)
	citOK, citIssues, citWarns := CheckCitations(md, refs)
	phCount, phIssues := CheckPlaceholders(md)
	return Report{
		LitReviewOK:       litOK,
		LitReviewIssues:   litIssues,
		CitationOK:        citOK,
		CitationIssues:    citIssues,
		CitationWarnings:  citWarns,
		PlaceholderCount:  phCount,
		PlaceholderIssues: phIssues,
	}
}

// Print 打印检测报告。
func (r Report) Print() {
	fmt.Println("=== 质量检测 ===")
	if r.LitReviewOK {
		fmt.Println("[综述] OK 论述式")
	} else {
		fmt.Println("[综述] FAIL 问题：")
		for _, i := range r.LitReviewIssues {
			fmt.Println("  - " + i)
		}
	}
	if r.CitationOK {
		fmt.Println("[引用] OK 无裸幻觉引用")
	} else {
		fmt.Println("[引用] FAIL 裸引用幻觉（未在 refs 且无占位符提示）：")
		for _, i := range r.CitationIssues {
			fmt.Println("  - " + i)
		}
	}
	if len(r.CitationWarnings) > 0 {
		fmt.Println("[引用] WARN 有提示的幻觉（LLM 已标注需补充）：")
		for _, i := range r.CitationWarnings {
			fmt.Println("  - " + i)
		}
	}
	if r.PlaceholderCount == 0 {
		fmt.Println("[占位符] OK 无残留")
	} else {
		fmt.Printf("[占位符] WARN 残留 %d 处（缺真实数据/文献，需人工补）：\n", r.PlaceholderCount)
		for _, i := range r.PlaceholderIssues {
			fmt.Println("  - " + i)
		}
	}
	if r.AllPass() {
		fmt.Println("=== 通过 ===")
	} else {
		fmt.Println("=== 失败，需人工检查 ===")
	}
}

// CheckLitReview 判断综述是否为论述式。
func CheckLitReview(md string) (bool, []string) {
	content := extractSection(md, "文献综述")
	if strings.TrimSpace(content) == "" {
		return false, []string{"综述内容为空或未找到"}
	}
	lines := strings.Split(content, "\n")
	nonEmpty := 0
	skeletonLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		nonEmpty++
		if strings.HasPrefix(line, "- ") && strings.Contains(line, "采用") && strings.Contains(line, "研究了《") {
			skeletonLines++
		}
	}
	var issues []string
	if nonEmpty > 0 && float64(skeletonLines)/float64(nonEmpty) > 0.5 {
		issues = append(issues, fmt.Sprintf("综述 %d/%d 行是骨架列表式", skeletonLines, nonEmpty))
	}
	if strings.Contains(content, "此处需补充该组文献") {
		issues = append(issues, "综述含骨架占位符「此处需补充该组文献」")
	}
	return len(issues) == 0, issues
}

// CheckCitations 检查正文引用是否都在 refs 中。
// 返回 (ok, 裸幻觉列表, 有提示的幻觉列表)。
func CheckCitations(md string, refs []cite.Reference) (bool, []string, []string) {
	authorSet := make(map[string]bool)
	for _, r := range refs {
		for _, a := range r.Authors {
			name := strings.TrimSpace(a)
			if name == "" {
				continue
			}
			authorSet[name] = true
			if idx := strings.Index(name, " "); idx > 0 {
				authorSet[name[:idx]] = true
			}
		}
	}
	body := md
	if idx := strings.Index(md, "## 参考文献"); idx > 0 {
		body = md[:idx]
	}
	re := regexp.MustCompile(`([\x{4e00}-\x{9fa5}]{2,4}|[A-Za-z]+)(?:等|et al\.?)?\((\d{4})\)`)
	matches := re.FindAllStringSubmatchIndex(body, -1)
	missing := make(map[string]bool)
	warns := make(map[string]bool)
	for _, m := range matches {
		author := body[m[2]:m[3]]
		year := body[m[4]:m[5]]
		if matchAuthor(author, authorSet) {
			continue
		}
		key := author + "(" + year + ")"
		// 匹配位置后 30 字符内若有"此处需补充"，视为有提示，WARN 不 FAIL
		tailEnd := m[1] + 30
		if tailEnd > len(body) {
			tailEnd = len(body)
		}
		tail := body[m[1]:tailEnd]
		if strings.Contains(tail, "此处需补充") {
			warns[key] = true
		} else {
			missing[key] = true
		}
	}
	var issues []string
	for k := range missing {
		issues = append(issues, "正文引用未在 refs 中找到且无占位符提示："+k)
	}
	var warnings []string
	for k := range warns {
		warnings = append(warnings, k)
	}
	return len(issues) == 0, issues, warnings
}

// matchAuthor 判断候选作者是否在 refs 中，允许候选带前缀或后缀"等"。
func matchAuthor(candidate string, authorSet map[string]bool) bool {
	candidate = strings.TrimSuffix(candidate, "等")
	if authorSet[candidate] {
		return true
	}
	runes := []rune(candidate)
	for i := 1; i < len(runes); i++ {
		sub := string(runes[i:])
		if len([]rune(sub)) < 2 {
			break
		}
		if authorSet[sub] {
			return true
		}
	}
	return false
}

// CheckPlaceholders 统计占位符残留。
func CheckPlaceholders(md string) (int, []string) {
	patterns := []string{
		"（此处需补充",
		"（此处需填入数据）",
		"（此处需插入回归结果）",
		"此处需补充该组文献",
	}
	var issues []string
	count := 0
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		for _, p := range patterns {
			if strings.Contains(line, p) {
				count++
				issues = append(issues, "行 "+strconv.Itoa(i+1)+" 含占位符：「"+p+"」")
				break
			}
		}
	}
	return count, issues
}

// extractSection 提取以 "## 前缀" 开头的章节内容，到下一个 "## " 为止。
func extractSection(md, prefix string) string {
	lines := strings.Split(md, "\n")
	var sb strings.Builder
	inSection := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if inSection {
				break
			}
			if strings.HasPrefix(line, "## "+prefix) {
				inSection = true
				continue
			}
		}
		if inSection {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
