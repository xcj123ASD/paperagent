package cite

import (
	"fmt"
	"strings"
)

// GBT7714 实现 GB/T 7714-2015 顺序编码制。
type GBT7714 struct{}

func (GBT7714) Name() string { return "GB/T 7714-2015" }

func (GBT7714) Format(refs []Reference) (string, error) {
	if len(refs) == 0 {
		return "", nil
	}
	var sb strings.Builder
	for i, r := range refs {
		line := formatOne(r)
		if line == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, line))
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func formatOne(r Reference) string {
	switch r.Type {
	case "article-journal":
		return formatJournal(r)
	case "book":
		return formatBook(r)
	case "chapter":
		return formatChapter(r)
	case "thesis":
		return formatThesis(r)
	case "web":
		return formatWeb(r)
	case "report":
		return formatReport(r)
	default:
		return formatJournal(r)
	}
}

// 期刊论文：[1] 作者. 题名[J]. 刊名, 年, 卷(期): 页码.
func formatJournal(r Reference) string {
	var sb strings.Builder
	sb.WriteString(formatAuthors(r))
	sb.WriteString(". ")
	sb.WriteString(r.Title)
	sb.WriteString("[J]. ")
	sb.WriteString(r.Container)
	if r.Year > 0 {
		sb.WriteString(", ")
		sb.WriteString(fmt.Sprint(r.Year))
	}
	if r.Volume != "" {
		sb.WriteString(", ")
		sb.WriteString(r.Volume)
	}
	if r.Issue != "" {
		sb.WriteString("(")
		sb.WriteString(r.Issue)
		sb.WriteString(")")
	}
	if r.Pages != "" {
		sb.WriteString(": ")
		sb.WriteString(r.Pages)
	}
	sb.WriteString(".")
	return sb.String()
}

// 专著：[1] 作者. 书名[M]. 出版地: 出版社, 年.
func formatBook(r Reference) string {
	var sb strings.Builder
	sb.WriteString(formatAuthors(r))
	sb.WriteString(". ")
	sb.WriteString(r.Title)
	sb.WriteString("[M]. ")
	if r.Publisher != "" {
		sb.WriteString(r.Publisher)
	}
	if r.Year > 0 {
		sb.WriteString(", ")
		sb.WriteString(fmt.Sprint(r.Year))
	}
	sb.WriteString(".")
	return sb.String()
}

// 析出文献：[1] 作者. 题名[C]//文集名. 出版社, 年: 页码.
func formatChapter(r Reference) string {
	var sb strings.Builder
	sb.WriteString(formatAuthors(r))
	sb.WriteString(". ")
	sb.WriteString(r.Title)
	sb.WriteString("[C]//")
	sb.WriteString(r.Container)
	sb.WriteString(". ")
	if r.Publisher != "" {
		sb.WriteString(r.Publisher)
	}
	if r.Year > 0 {
		sb.WriteString(", ")
		sb.WriteString(fmt.Sprint(r.Year))
	}
	if r.Pages != "" {
		sb.WriteString(": ")
		sb.WriteString(r.Pages)
	}
	sb.WriteString(".")
	return sb.String()
}

// 学位论文：[1] 作者. 题名[D]. 地点: 单位, 年.
func formatThesis(r Reference) string {
	var sb strings.Builder
	sb.WriteString(formatAuthors(r))
	sb.WriteString(". ")
	sb.WriteString(r.Title)
	sb.WriteString("[D]. ")
	if r.Publisher != "" {
		sb.WriteString(r.Publisher)
	}
	if r.Year > 0 {
		sb.WriteString(", ")
		sb.WriteString(fmt.Sprint(r.Year))
	}
	sb.WriteString(".")
	return sb.String()
}

// 电子资源：[1] 作者. 题名[EB/OL]. (发布日期)[访问日期]. URL.
func formatWeb(r Reference) string {
	var sb strings.Builder
	sb.WriteString(formatAuthors(r))
	sb.WriteString(". ")
	sb.WriteString(r.Title)
	sb.WriteString("[EB/OL]. ")
	if r.Year > 0 {
		sb.WriteString(fmt.Sprintf("(%d)", r.Year))
	}
	if r.AccessDate != "" {
		sb.WriteString("[")
		sb.WriteString(r.AccessDate)
		sb.WriteString("]. ")
	}
	if r.URL != "" {
		sb.WriteString(r.URL)
	}
	sb.WriteString(".")
	return sb.String()
}

// 报告：[1] 作者. 题名[R]. 地点: 机构, 年.
func formatReport(r Reference) string {
	var sb strings.Builder
	sb.WriteString(formatAuthors(r))
	sb.WriteString(". ")
	sb.WriteString(r.Title)
	sb.WriteString("[R]. ")
	if r.Publisher != "" {
		sb.WriteString(r.Publisher)
	}
	if r.Year > 0 {
		sb.WriteString(", ")
		sb.WriteString(fmt.Sprint(r.Year))
	}
	sb.WriteString(".")
	return sb.String()
}

// formatAuthors 中文作者原样，英文作者姓前名缩写，>3 人加"等"/"et al."
func formatAuthors(r Reference) string {
	if len(r.Authors) == 0 {
		return "佚名"
	}
	var names []string
	if r.Lang == "en" {
		for _, a := range r.Authors {
			names = append(names, formatEnglishAuthor(a))
		}
		if len(names) > 3 {
			return strings.Join(names[:3], ", ") + ", et al"
		}
		return strings.Join(names, ", ")
	}
	names = append(names, r.Authors...)
	if len(names) > 3 {
		return strings.Join(names[:3], ", ") + ", 等"
	}
	return strings.Join(names, ", ")
}

// formatEnglishAuthor 支持三种输入格式：
//   - "John Smith"（名 + 姓）→ "Smith J"
//   - "Smith, John"（姓, 名）→ "Smith J"
//   - "Smith J" / "Goldfarb A"（姓 + 首字母，OpenAlex 格式）→ "Smith J"
func formatEnglishAuthor(a string) string {
	a = strings.TrimSpace(a)
	if a == "" {
		return ""
	}

	// 格式 2：姓, 名
	if strings.Contains(a, ",") {
		parts := strings.SplitN(a, ",", 2)
		surname := strings.TrimSpace(parts[0])
		given := strings.TrimSpace(parts[1])
		var initials []string
		for _, w := range strings.Fields(given) {
			if w != "" {
				initials = append(initials, string([]rune(w)[0]))
			}
		}
		if len(initials) == 0 {
			return surname
		}
		return surname + " " + strings.Join(initials, " ")
	}

	fields := strings.Fields(a)
	if len(fields) == 1 {
		return a
	}

	// 格式 3：姓 + 首字母（最后一个字段全是大写单字母或点号分隔首字母）
	last := fields[len(fields)-1]
	if isInitials(last) {
		surname := strings.Join(fields[:len(fields)-1], " ")
		return surname + " " + normalizeInitials(last)
	}

	// 格式 1：名 + 姓
	surname := fields[len(fields)-1]
	var initials []string
	for _, w := range fields[:len(fields)-1] {
		if w != "" {
			initials = append(initials, string([]rune(w)[0]))
		}
	}
	if len(initials) == 0 {
		return surname
	}
	return surname + " " + strings.Join(initials, " ")
}

// isInitials 判断字段是否是首字母形式（如 "A"、"J.R."、"JR"）
func isInitials(s string) bool {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// normalizeInitials 把 "A" / "J.R." / "JR" 统一成 "A" / "J R" / "J R"
func normalizeInitials(s string) string {
	s = strings.ReplaceAll(s, ".", "")
	var out []string
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			out = append(out, string(r))
		}
	}
	return strings.Join(out, " ")
}