package litreview

import (
	"fmt"
	"sort"
	"strings"

	"paperagent/internal/cite"
)

// RenderSkeleton 从参考文献生成文献综述骨架，永不返回空字符串。
//
// 分层输出：
//   - 有 Conclusion 或 Abstract 的文献：输出带论述的句子
//     "X(年份)采用【方法】研究了《标题》，发现【结论】"
//   - 只有元数据的文献：回落一句"X(年份)研究了《标题》"
//
// 每组末尾保留"（此处需补充...）"占位，提示人工补组间关系。
func RenderSkeleton(refs []cite.Reference) string {
	if len(refs) == 0 {
		return "（此处需补充文献综述：请先在 refs.json 中录入参考文献。）"
	}

	groups := make(map[int][]cite.Reference)
	for _, r := range refs {
		groups[r.Year] = append(groups[r.Year], r)
	}

	years := make([]int, 0, len(groups))
	for y := range groups {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	var b strings.Builder
	b.WriteString("既有研究围绕本文主题已形成若干积累，可按年份梳理如下。\n\n")
	for _, y := range years {
		fmt.Fprintf(&b, "%d 年：\n", y)
		for _, r := range groups[y] {
			b.WriteString("- ")
			b.WriteString(formatRef(r))
			b.WriteString("\n")
		}
		b.WriteString("（此处需补充该组文献的具体发现、方法与相互关系。）\n\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// formatRef 把单条文献格式化成带论述的句子。
//
// 优先级：
//  1. 有 Conclusion：输出"X(年份)采用【方法】研究了《标题》，发现【结论】"
//  2. 只有 Abstract：输出"X(年份)研究了《标题》，摘要指出【摘要】"
//  3. 只有元数据：输出"X(年份)研究了《标题》"
//
// Relation 有值时追加"（与本文关系：X）"。
func formatRef(r cite.Reference) string {
	author := formatAuthors(r.Authors)
	year := "n.d."
	if r.Year > 0 {
		year = fmt.Sprintf("%d", r.Year)
	}
	title := r.Title
	if title == "" {
		title = "（无标题）"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s(%s)", author, year)

	if r.Method != "" {
		fmt.Fprintf(&b, "采用%s", r.Method)
	}
	fmt.Fprintf(&b, "研究了《%s》", title)

	if r.Conclusion != "" {
		fmt.Fprintf(&b, "，发现%s", strings.TrimRight(r.Conclusion, "。."))
	} else if r.Abstract != "" {
		fmt.Fprintf(&b, "，摘要指出%s", strings.TrimRight(r.Abstract, "。."))
	}

	b.WriteString("。")

	if r.Relation != "" {
		fmt.Fprintf(&b, "（与本文关系：%s）", r.Relation)
	}
	return b.String()
}

// formatAuthors 格式化作者列表：1 人直写，多人取第一作者 + "等"。
func formatAuthors(authors []string) string {
	if len(authors) == 0 {
		return "佚名"
	}
	if len(authors) == 1 {
		return authors[0]
	}
	return authors[0] + "等"
}
