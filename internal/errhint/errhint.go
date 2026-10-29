package errhint

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"paperagent/internal/llm"
)

// Explain 把错误翻译成「原因 + 怎么办」的中文提示。
// 无法识别时返回空字符串，调用方自行打印原始 err。
func Explain(err error) string {
	if err == nil {
		return ""
	}

	// 1. HTTP 错误（LLM 返回非 200）
	var he *llm.HTTPError
	if errors.As(err, &he) {
		return explainHTTP(he)
	}

	// 2. context 超时/取消
	if errors.Is(err, context.DeadlineExceeded) {
		return "超时：LLM 响应超过 90 秒。检查网络，或减少 --refs 数量后重试。"
	}
	if errors.Is(err, context.Canceled) {
		return "已取消。"
	}

	// 3. 字符串模式匹配
	msg := err.Error()
	switch {
	case strings.Contains(msg, "未找到 pandoc"):
		return "导出失败：未找到 pandoc。运行 paperagent --install-pandoc 自动安装，或 winget install JohnMacFarlane.Pandoc。"
	case strings.Contains(msg, "PANDOC_PATH"):
		return "导出失败：PANDOC_PATH 指向的文件不存在。检查 setx PANDOC_PATH 路径是否正确。"
	case strings.Contains(msg, "invalid --mode"):
		return "参数错误：--mode 只能是 thesis 或 journal。"
	case strings.Contains(msg, "生成内容过短"):
		if strings.Contains(msg, "（0 字") || strings.Contains(msg, "(0 字") {
			return "LLM 返回空内容。检查：1) Key 是否有效（echo $env:DEEPSEEK_API_KEY 应输出 35 位）；2) DeepSeek 是否欠费（HTTP 402）；3) 是否被 FallbackClient 切到智谱 glm-5.3（推理模型易返回空），可临时 unset ZAI_API_KEY 强制走 DeepSeek。"
		}
		return "LLM 返回内容过短。可能原因：prompt 太长导致模型截断。减少 --refs 数量，或缩短 design.json 描述后重试。"
	case strings.Contains(msg, "LLM 润色失败或过短"):
		return "LLM 润色失败或返回过短。检查 Key、网络、账户余额；若用智谱 glm-5.3，考虑切回 DeepSeek：unset ZAI_API_KEY。"
	case strings.Contains(msg, "token 用量已达上限"):
		return "成本配额：今日 token 用量已达 --max-tokens-per-day 上限。明日再跑，或提高上限（0 表示不限）。"
	case strings.Contains(msg, "load design failed"):
		return "研究设计文件读取失败：检查 design.json 路径是否正确、JSON 格式是否合法。"
	case strings.Contains(msg, "load refs"):
		return "参考文献文件读取失败：检查 refs.json 路径和 JSON 格式。"
	case strings.Contains(msg, "write output failed"):
		return "写出文件失败：检查输出目录是否有写权限，磁盘是否已满。"
	case strings.Contains(msg, "marshal input failed"):
		return "内部错误：序列化输入失败，请提 issue 并附日志。"
	case strings.Contains(msg, "step1 failed"):
		return "step1 失败：题材/期刊匹配出错，检查 topic 参数。"
	case strings.Contains(msg, "step2 failed") || strings.Contains(msg, "step2_frame not registered"):
		return "step2 失败：章节结构生成出错，请提 issue。"
	case strings.Contains(msg, "step3 failed") || strings.Contains(msg, "step3_litreview not registered"):
		return "step3 失败：正文生成出错，检查 LLM 是否可用（运行 . .\\load.ps1 确认 Key）。"
	case strings.Contains(msg, "parse frame failed"):
		return "内部错误：解析章节结构失败，请提 issue。"
	case strings.Contains(msg, "export output missing"):
		return "导出失败：pandoc 声称成功但输出文件不存在，检查磁盘和权限。"
	case strings.Contains(msg, "export failed"):
		return "导出失败：pandoc 执行出错，检查 pandoc 是否可运行（pandoc --version）。"
	case strings.Contains(msg, "安装 pandoc 失败"):
		return "pandoc 安装失败：按上方方案 A/B/C 手动安装。"
	}

	return ""
}

func explainHTTP(he *llm.HTTPError) string {
	body := strings.TrimSpace(he.Body)
	if len(body) > 200 {
		body = body[:200] + "..."
	}
	switch he.StatusCode {
	case 401:
		return "认证失败（HTTP 401）：API Key 无效。检查 setx DEEPSEEK_API_KEY \"你的Key\" 是否配置正确，Key 是否过期。"
	case 402:
		return "余额不足（HTTP 402）：DeepSeek 账户欠费。去 https://platform.deepseek.com/ 充值。"
	case 403:
		return "禁止访问（HTTP 403）：Key 权限不足或被封。检查账户状态。"
	case 404:
		return fmt.Sprintf("接口不存在（HTTP 404）：检查 BaseURL 是否写错。响应：%s", body)
	case 429:
		return "限流（HTTP 429）：请求太频繁。等 1 分钟重试，或减少并发。"
	}
	if he.StatusCode >= 500 {
		return fmt.Sprintf("服务端故障（HTTP %d）：LLM 服务商故障。换个时间重试，或切智谱：setx LLM_PROVIDER zhipu。响应：%s", he.StatusCode, body)
	}
	return fmt.Sprintf("LLM 返回错误（HTTP %d）：%s", he.StatusCode, body)
}
