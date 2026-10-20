package cite

// Formatter 参考文献格式化器。
// 接口保持稳定：输入标准化 Reference，输出可直接插入论文的文本。
// 内部实现可替换（GB/T 7714 / APA / 未来从 SKILL.md 加载模板），调用方不感知。
type Formatter interface {
// Format 把引用列表渲染成完整参考文献段落，条目之间空行分隔。
Format(refs []Reference) (string, error)

// Name 返回格式名，用于日志和用户提示。
Name() string
}

// SortReferences 按作者姓氏 + 年份排序，供各 Formatter 复用。
// 中文作者按拼音排序暂不处理，先按原顺序，M2 之后再补。
func SortReferences(refs []Reference) []Reference {
out := make([]Reference, len(refs))
copy(out, refs)
return out
}

// normalizeYear 缺失年份时返回 0，供 Formatter 判断是否需要占位符。
func normalizeYear(y int) int {
if y < 0 {
return 0
}
return y
}