package cite

// Merge 用 src 中的非空字段补全 dst 的空字段，不覆盖已填内容。
func Merge(dst *Reference, src Reference) {
if dst.Title == "" {
dst.Title = src.Title
}
if dst.Year == 0 {
dst.Year = src.Year
}
if dst.Container == "" {
dst.Container = src.Container
}
if dst.Volume == "" {
dst.Volume = src.Volume
}
if dst.Issue == "" {
dst.Issue = src.Issue
}
if dst.Pages == "" {
dst.Pages = src.Pages
}
if dst.DOI == "" {
dst.DOI = src.DOI
}
if len(dst.Authors) == 0 {
dst.Authors = src.Authors
}
if dst.Lang == "" {
dst.Lang = src.Lang
}
if dst.Type == "" {
dst.Type = src.Type
}
}