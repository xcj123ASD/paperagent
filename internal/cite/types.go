package cite

// Reference 一条参考文献的标准化结构。
// 字段命名对齐 CSL-JSON（Citation Style Language），便于未来接 Zotero / Crossref。
type Reference struct {
	// 类型：article-journal / book / chapter / thesis / web/ report
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

	// ---- L3 深度字段（可选，填了综述有论述，不填回落骨架）----

	// Abstract 摘要原文（100–300 字，用于综述提炼观点）
	Abstract string

	// Conclusion 核心结论一句话（如"数字化转型显著提升企业创新产出"）
	Conclusion string

	// Method 研究方法（如"DID"、"固定效应"、"PSM-DID"、"案例研究"）
	Method string

	// Relation 与本文的关系（如"支持"、"反驳"、"补充"、"方法借鉴"）
	Relation string
}
