package cite

import "context"

// VerifyStatus 校验结果状态。
type VerifyStatus int

const (
	// VerifyFound 在 OpenAlex 找到匹配记录。
	VerifyFound VerifyStatus = iota
	// VerifyNotFound 查询成功但无匹配。
	VerifyNotFound
	// VerifyError 网络或解析错误，调用方应视为"暂不可校验"而非"文献不存在"。
	VerifyError
)

// VerifyResult 单条文献的校验结果。
type VerifyResult struct {
	Status VerifyStatus

	// Matched 从 OpenAlex 拉回的标准化元数据，可用于补全缺失字段。
	// Status != VerifyFound 时为 nil。
	Matched *Reference

	// Confidence 0–1。DOI 精确匹配为 1.0；标题匹配按相似度给分。
	Confidence float64

	// Message 人类可读的说明，用于日志和用户提示。
	Message string
}

// Verifier 文献真实性校验器。
// 只做单向校验：用户给 Reference，Verifier 查是否存在并返回标准化元数据。
// 不做全库检索（那是检索，不是校验）。
type Verifier interface {
	Verify(ctx context.Context, ref Reference) (VerifyResult, error)
	Name() string
}
