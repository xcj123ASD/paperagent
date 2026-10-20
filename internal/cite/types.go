package cite

// Reference 一条参考文献的标准化结构。
// 字段命名对齐 CSL-JSON（Citation Style Language），便于未来接 Zotero / Crossref。
type Reference struct {
// 类型：article-journal / book / chapter / thesis / web / report
Type string

// 作者，按"姓, 名"或中文姓名原样存
Authors []string

// 标题
Title string

// 期刊名 / 书名 / 会议名
Container string

// 出版年
Year int

// 卷、期、页码
Volume string
Issue  string
Pages  string

// 出版社 / 学位授予单位
Publisher string

// DOI（不带 https://doi.org/ 前缀）
DOI string

// URL（DOI 缺失时使用）
URL string

// 访问日期，格式 2006-01-02，仅 web 类必填
AccessDate string

// 语言：zh / en
Lang string
}