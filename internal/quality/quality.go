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
	CitationWarnings  []string // 有提示的幻觉/拼写变形，判 WARN
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
		fmt.Println("[引用] WARN 有提示的幻觉/拼写变形（可识别，需人工确认）：")
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
// 返回 (ok, 裸幻觉列表, 有提示的幻觉/变形列表)。
func CheckCitations(md string, refs []cite.Reference) (bool, []string, []string) {
	authorSet := make(map[string]bool)
	var authorList []string
	for _, r := range refs {
		for _, a := range r.Authors {
			name := strings.TrimSpace(a)
			if name == "" {
				continue
			}
			authorList = append(authorList, name)
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
	re := regexp.MustCompile(`((?:[A-Z][A-Za-z]+(?:\s+[A-Z]\.?)*)|[\x{4e00}-\x{9fa5}]{2,4})(?:等|et al\.?)?\((\d{4})\)`)
	matches := re.FindAllStringSubmatchIndex(body, -1)
	missing := make(map[string]bool)
	warns := make(map[string]bool)
	for _, m := range matches {
		author := body[m[2]:m[3]]
		if fields := strings.Fields(author); len(fields) > 1 {
			author = fields[0]
		}
		year := body[m[4]:m[5]]
		level := matchAuthorLevel(author, authorSet, authorList)
		if level == 0 {
			continue
		}
		key := author + "(" + year + ")"
		tailEnd := m[1] + 30
		if tailEnd > len(body) {
			tailEnd = len(body)
		}
		tail := body[m[1]:tailEnd]
		if level == 1 || strings.Contains(tail, "此处需补充") {
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

// matchAuthorLevel 返回匹配级别：0=完全匹配，1=模糊匹配（WARN），2=不匹配（FAIL）。
func matchAuthorLevel(candidate string, authorSet map[string]bool, authorList []string) int {
	candidate = strings.TrimSuffix(candidate, "等")
	if authorSet[candidate] {
		return 0
	}
	runes := []rune(candidate)
	// 去前缀子串匹配
	for i := 1; i < len(runes); i++ {
		sub := string(runes[i:])
		if len([]rune(sub)) < 2 {
			break
		}
		if authorSet[sub] {
			return 0
		}
	}
	// 末尾 1-2 字子串（处理正则贪婪把前文吃进候选）
	for i := 1; i <= 2 && i < len(runes); i++ {
		sub := string(runes[len(runes)-i:])
		if authorSet[sub] {
			return 0
		}
	}
	// 英文拼写变形：前 5 字符相同或编辑距离 ≤2
	if isEnglish(candidate) && len(candidate) >= 5 {
		for _, a := range authorList {
			if !isEnglish(a) || len(a) < 5 {
				continue
			}
			if strings.EqualFold(candidate[:5], a[:5]) {
				return 1
			}
			if levenshtein(candidate, a) <= 2 {
				return 1
			}
		}
	}
	// 中文候选：首字（姓）或末尾单字可能是英文姓的中文翻译
	if len(runes) >= 1 {
		var checks []rune
		if isChinese(runes[0]) {
			checks = append(checks, runes[0])
		}
		last := runes[len(runes)-1]
		if isChinese(last) && last != runes[0] {
			checks = append(checks, last)
		}
		for _, r := range checks {
			ch := string(r)
			for _, a := range authorList {
				if !isEnglish(a) {
					continue
				}
				fields := strings.Fields(a)
				if len(fields) == 0 {
					continue
				}
				if pinyinToSurname(fields[0]) == ch {
					return 1
				}
			}
		}
	}
	return 2
}

func isEnglish(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func isChinese(r rune) bool {
	return r >= 0x4e00 && r <= 0x9fa5
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
	}
	for i := 0; i <= la; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			dp[i][j] = min3(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	return dp[la][lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// pinyinToSurname 常见中文姓氏的拼音映射，用于识别 LLM 把英文姓翻译成中文单字的情况。
func pinyinToSurname(pinyin string) string {
	m := map[string]string{
		"liu": "刘", "wang": "王", "li": "李", "zhang": "张",
		"chen": "陈", "yang": "杨", "huang": "黄", "zhao": "赵",
		"wu": "吴", "zhou": "周", "xu": "徐", "sun": "孙",
		"ma": "马", "zhu": "朱", "hu": "胡", "guo": "郭",
		"he": "何", "gao": "高", "lin": "林", "luo": "罗",
		"zheng": "郑", "liang": "梁", "xie": "谢", "song": "宋",
		"tang": "唐", "han": "韩", "feng": "冯", "deng": "邓",
		"cao": "曹", "peng": "彭", "xiao": "肖", "tian": "田",
		"dong": "董", "pan": "潘", "cai": "蔡", "jiang": "蒋",
		"yu": "余", "du": "杜", "ye": "叶", "cheng": "程",
		"su": "苏", "wei": "魏", "ding": "丁", "ren": "任",
		"shen": "沈", "yao": "姚", "lu": "卢", "cui": "崔",
		"zhong": "钟", "tan": "谭", "fan": "范", "jin": "金",
		"shi": "石", "liao": "廖", "jia": "贾", "xia": "夏",
		"fu": "傅", "fang": "方", "zou": "邹", "xiong": "熊",
		"bai": "白", "meng": "孟", "qin": "秦", "qiu": "邱",
		"hou": "侯", "yin": "尹", "xue": "薛", "yan": "闫",
		"duan": "段", "lei": "雷", "long": "龙", "tao": "陶",
		"mao": "毛", "hao": "郝", "gu": "顾",
		"gong": "龚", "shao": "邵", "wan": "万", "qian": "钱",
		"dai": "戴", "mo": "莫", "kong": "孔",
		"xiang": "向", "chang": "常", "wen": "温", "kang": "康",
		"niu": "牛", "ge": "葛",
	}
	if v, ok := m[strings.ToLower(pinyin)]; ok {
		return v
	}
	return ""
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
