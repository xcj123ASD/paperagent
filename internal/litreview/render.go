package litreview

import (
	"fmt"
	"sort"
	"strings"

	"paperagent/internal/cite"
)

// RenderSkeleton 从参考文献生成文献综述骨架，永不返回空字符串。
// 按年份降序分组，每篇一句"作者(年份)研究了《标题》"，每组后附可执行占位。
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

// formatRef 把单条文献格式化成 "作者(年份)研究了《标题》"。
func formatRef(r cite.Reference) string {
	author := "佚名"
	if len(r.Authors) > 0 {
		if len(r.Authors) == 1 {
			author = r.Authors[0]
		} else {
			author = r.Authors[0] + "等"
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
	return fmt.Sprintf("%s(%s)研究了《%s》", author, year, title)
}
