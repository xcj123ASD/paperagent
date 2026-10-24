package browse

import "context"

// Browser 是浏览器操作的抽象接口。
//
// 设计原则：
//   - 只暴露只读能力（Navigate + ReadPage），写操作（click/fill/download）
//     由后续阶段单独定义接口，不混在这里
//   - 不暴露原始 JS 执行能力（无 Evaluate 方法），从接口层阻断 RCE
//   - 所有方法带 context，便于超时控制
//
// 实现方：
//   - MockBrowser：测试用，不碰网络
//   - PlaywrightBrowser：阶段 2 实现，走 MCP client
type Browser interface {
	// Navigate 打开指定 URL。
	// 调用方必须先过 ValidateResolvedURL，接口本身不做 SSRF 校验——
	// 职责分离：校验是安全层，执行是浏览器层。
	Navigate(ctx context.Context, rawURL string) error

	// ReadPage 返回当前页面正文（纯文本，已剥离 HTML 标签）。
	// 页面未加载时返回错误。
	ReadPage(ctx context.Context) (string, error)

	// Close 释放浏览器资源。可重复调用，幂等。
	Close() error
}

// PageContent 是 ReadPage 的结构化返回（预留扩展）。
// 当前 ReadPage 只返回 string，未来若要标题/链接列表可升级用此结构。
type PageContent struct {
	URL   string
	Title string
	Text  string
}
