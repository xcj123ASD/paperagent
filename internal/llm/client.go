package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"paperagent/internal/model"
)

type KeyPointRequest struct {
	Topic       string
	Section     string
	Role        string
	ArticleType string
	MaxPoints   int
	Mode        model.Mode
}

type ExpandSectionRequest struct {
	Topic            string
	Section          string
	Role             string
	KeyPoints        []string
	WordBudget       int
	ArticleType      string
	PreviousSummary  string
	RequiredEvidence []string
	Forbidden        []string
	Mode             model.Mode
	DesignContext    string
	References       []string
}

type Client interface {
	KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error)
	ExpandSection(ctx context.Context, req ExpandSectionRequest) (string, error)
}

// UsageReporter 可选接口：实现者可返回累计 token 用量。MockClient 不实现。
type UsageReporter interface {
	Usage() (input, output int64)
	ResetUsage()
}

type MockClient struct{}

func (MockClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	return []string{
		req.Section + "：要点一（Mock）",
		req.Section + "：要点二（Mock）",
	}, nil
}

func (MockClient) ExpandSection(ctx context.Context, req ExpandSectionRequest) (string, error) {
	para := fmt.Sprintf("【%s】这是 Mock 生成的正文段落，覆盖要点：%s。",
		req.Section, strings.Join(req.KeyPoints, "、"))
	paraRunes := len([]rune(para))
	repeat := req.WordBudget / (paraRunes + 1)
	if repeat < 1 {
		repeat = 1
	}
	parts := make([]string, 0, repeat)
	for i := 0; i < repeat; i++ {
		parts = append(parts, para)
	}
	return strings.Join(parts, "\n\n"), nil
}

type FallbackClient struct {
	Primary Client
	Backup  Client
}

func (f FallbackClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	points, err := f.Primary.KeyPoints(ctx, req)
	if err == nil {
		return points, nil
	}
	if f.Backup == nil {
		return nil, err
	}
	return f.Backup.KeyPoints(ctx, req)
}

func (f FallbackClient) ExpandSection(ctx context.Context, req ExpandSectionRequest) (string, error) {
	content, err := f.Primary.ExpandSection(ctx, req)
	if err == nil {
		return content, nil
	}
	if f.Backup == nil {
		return "", err
	}
	return f.Backup.ExpandSection(ctx, req)
}

// Usage 转发 Primary 的用量；Primary 不支持时返回 0。
func (f FallbackClient) Usage() (int64, int64) {
	if u, ok := f.Primary.(UsageReporter); ok {
		return u.Usage()
	}
	return 0, 0
}

// ResetUsage 转发 Primary 的重置。
func (f FallbackClient) ResetUsage() {
	if u, ok := f.Primary.(UsageReporter); ok {
		u.ResetUsage()
	}
}

type OpenAICompatClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client

	mu           sync.Mutex
	inputTokens  int64
	outputTokens int64
}

func NewOpenAICompat(baseURL, apiKey, model string) *OpenAICompatClient {
	return &OpenAICompatClient{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// Usage 返回累计输入/输出 token 数（线程安全）。
func (c *OpenAICompatClient) Usage() (int64, int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inputTokens, c.outputTokens
}

// ResetUsage 清零累计用量。
func (c *OpenAICompatClient) ResetUsage() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inputTokens = 0
	c.outputTokens = 0
}

func (c *OpenAICompatClient) addUsage(in, out int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inputTokens += in
	c.outputTokens += out
}

func NewFromEnv() Client {
	if os.Getenv("LLM_PROVIDER") == "mock" {
		return MockClient{}
	}
	zhipuKey := os.Getenv("ZAI_API_KEY")
	if zhipuKey == "" {
		zhipuKey = os.Getenv("ZHIPU_API_KEY")
	}
	dsKey := os.Getenv("DEEPSEEK_API_KEY")

	var zhipu, deepseek Client
	if zhipuKey != "" {
		zhipu = NewOpenAICompat(
			"https://open.bigmodel.cn/api/paas/v4/",
			zhipuKey,
			"glm-5.3",
		)
	}
	if dsKey != "" {
		deepseek = NewOpenAICompat(
			"https://api.deepseek.com",
			dsKey,
			"deepseek-chat",
		)
	}

	switch {
	case zhipu != nil && deepseek != nil:
		if os.Getenv("LLM_PROVIDER") == "zhipu" {
			return FallbackClient{Primary: zhipu, Backup: deepseek}
		}
		return FallbackClient{Primary: deepseek, Backup: zhipu}
	case zhipu != nil:
		return zhipu
	case deepseek != nil:
		return deepseek
	default:
		return nil
	}
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("LLM 返回 %d: %s", e.StatusCode, e.Body)
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if he, ok := err.(*HTTPError); ok {
		if he.StatusCode >= 400 && he.StatusCode < 500 && he.StatusCode != 429 {
			return false
		}
	}
	return true
}

func buildKeyPointPrompt(req KeyPointRequest) string {
	return fmt.Sprintf(
		"你是一位学术论文写作助手。请为以下章节生成不超过 %d 个核心要点，每个要点一行，不要编号，不要解释。\n"+
			"论文主题：%s\n章节：%s\n章节作用：%s\n文章类型：%s",
		req.MaxPoints, req.Topic, req.Section, req.Role, req.ArticleType,
	)
}

// needsDataSection 判断章节是否属于“需要真实数据支撑”的类型。
func needsDataSection(name string) bool {
	keywords := []string{"研究设计", "实证结果", "数据分析", "研究方法", "模型设定", "变量测度"}
	for _, k := range keywords {
		if strings.Contains(name, k) {
			return true
		}
	}
	return false
}

func buildExpandSectionPrompt(req ExpandSectionRequest) string {
	var sb strings.Builder
	needsData := needsDataSection(req.Section)

	if req.Mode == model.ModeJournal {
		sb.WriteString(fmt.Sprintf("你是一位学术期刊论文写作助手。请为以下章节撰写学术正文，严格不超过 %d 字，超出部分会被系统截断。\n", req.WordBudget))
		sb.WriteString(fmt.Sprintf("论文主题：%s\n", req.Topic))
		sb.WriteString(fmt.Sprintf("章节：%s\n", req.Section))
		sb.WriteString(fmt.Sprintf("章节作用：%s\n", req.Role))
		sb.WriteString(fmt.Sprintf("文章类型：%s\n", req.ArticleType))
		if len(req.KeyPoints) > 0 {
			sb.WriteString("本章节需要覆盖的要点：\n")
			for _, p := range req.KeyPoints {
				sb.WriteString("- " + p + "\n")
			}
		}
		if len(req.References) > 0 {
			sb.WriteString("【可引用文献】（引用时用 作者(年份) 格式，不得虚构。作者姓名必须与下列列表逐字一致，不得拼写错误，如 Brynjolfsson 不能写成 Brynjansson）：\n")
			for _, r := range req.References {
				sb.WriteString("- " + r + "\n")
			}
		}
		if req.PreviousSummary != "" {
			sb.WriteString(fmt.Sprintf("前文摘要（避免重复，保持衔接）：%s\n", req.PreviousSummary))
		}
		if len(req.RequiredEvidence) > 0 {
			sb.WriteString(fmt.Sprintf("必须包含的证据/表格：%s\n", strings.Join(req.RequiredEvidence, "、")))
		}
		if len(req.Forbidden) > 0 {
			sb.WriteString(fmt.Sprintf("禁止出现的内容：%s\n", strings.Join(req.Forbidden, "、")))
		}
		sb.WriteString("要求：\n")
		sb.WriteString("1. 学术书面语，论证严密，逻辑递进，避免罗列；\n")
		sb.WriteString("2. 体现理论对话：引用既有研究时需指出其贡献与局限，不要简单堆砌；\n")
		sb.WriteString("3. 不做无据推断，不编造具体数据或虚构文献；但若需引用而缺来源，用占位符标注（如「（此处需补充文献）」），不要返回空内容；\n")
		sb.WriteString("4. 段落分明，不要出现小标题。\n")
		sb.WriteString("5. 研究假设必须与本文实证设计严格一致：只推导设计中包含的中介变量与分组维度，不要提出设计中不检验的假设；若设计为分组异质性分析，请用“异质性”表述而非“调节”。\n")
		if needsData {
			sb.WriteString("\n本章节属于需要真实数据支撑的类型。即使没有真实数据，也必须写出完整的分析框架，不得返回空内容：\n")
			sb.WriteString("- 若为研究设计：写出数据来源说明、变量定义（变量名、符号、定义、测度方式）、模型设定（公式或方程形式）、识别策略；\n")
			sb.WriteString("- 若为实证结果：写出描述性统计框架、基准回归表结构（列名、变量名、预期方向）、稳健性检验思路、内生性处理方案；\n")
			sb.WriteString("具体数字用「（此处需填入数据）」占位，但框架、论证逻辑、文字表述必须完整。\n")
		}
		if req.DesignContext != "" {
			sb.WriteString("\n")
			sb.WriteString(req.DesignContext)
			sb.WriteString("\n请严格基于以上研究设计输入撰写本章节，不要虚构与输入不符的数据、变量名或方法。\n")
		}
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("你是一位学术论文写作助手。请为以下章节撰写正文，严格不超过 %d 字，超出部分会被系统截断。\n", req.WordBudget))
	sb.WriteString(fmt.Sprintf("论文主题：%s\n", req.Topic))
	sb.WriteString(fmt.Sprintf("章节：%s\n", req.Section))
	sb.WriteString(fmt.Sprintf("章节作用：%s\n", req.Role))
	sb.WriteString(fmt.Sprintf("文章类型：%s\n", req.ArticleType))
	if len(req.KeyPoints) > 0 {
		sb.WriteString("本章节需要覆盖的要点：\n")
		for _, p := range req.KeyPoints {
			sb.WriteString("- " + p + "\n")
		}
	}
	if len(req.References) > 0 {
		sb.WriteString("【可引用文献】（引用时用 作者(年份) 格式，不得虚构。作者姓名必须与下列列表逐字一致，不得拼写错误）：\n")
		for _, r := range req.References {
			sb.WriteString("- " + r + "\n")
		}
	}
	if req.PreviousSummary != "" {
		sb.WriteString(fmt.Sprintf("前文摘要（避免重复，保持衔接）：%s\n", req.PreviousSummary))
	}
	if len(req.RequiredEvidence) > 0 {
		sb.WriteString(fmt.Sprintf("必须包含的证据/表格：%s\n", strings.Join(req.RequiredEvidence, "、")))
	}
	if len(req.Forbidden) > 0 {
		sb.WriteString(fmt.Sprintf("禁止出现的内容：%s\n", strings.Join(req.Forbidden, "、")))
	}
	sb.WriteString("要求：学术书面语，段落分明，不要出现小标题。\n")
	sb.WriteString("重要：如需引用文献或报告数据但你没有真实来源，请用占位符标注，例如「（此处需补充具体文献）」「（此处需插入回归结果）」，不要虚构具体文献、作者或数据。宁可留占位符，也不要返回空内容。\n")
	if needsData {
		sb.WriteString("\n本章节属于需要真实数据支撑的类型。即使没有真实数据，也必须写出完整的分析框架，不得返回空内容：\n")
		sb.WriteString("- 若为研究设计：写出数据来源说明、变量定义（变量名、符号、定义、测度方式）、模型设定（公式或方程形式）、识别策略；\n")
		sb.WriteString("- 若为实证结果：写出描述性统计框架、基准回归表结构（列名、变量名、预期方向）、稳健性检验思路、内生性处理方案；\n")
		sb.WriteString("具体数字用「（此处需填入数据）」占位，但框架、论证逻辑、文字表述必须完整。\n")
	}
	if req.DesignContext != "" {
		sb.WriteString("\n")
		sb.WriteString(req.DesignContext)
		sb.WriteString("\n请严格基于以上研究设计输入撰写本章节，不要虚构与输入不符的数据、变量名或方法。\n")
	}
	return sb.String()
}

func (c *OpenAICompatClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		points, err := c.doKeyPoints(ctx, req)
		if err == nil {
			return points, nil
		}
		lastErr = err
		if !isRetryable(err) || attempt == maxAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	return nil, fmt.Errorf("重试 %d 次后仍失败: %w", maxAttempts, lastErr)
}

func (c *OpenAICompatClient) doKeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	content, err := c.chat(ctx, buildKeyPointPrompt(req), 512)
	if err != nil {
		return nil, err
	}
	var points []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "0123456789.、 ）)-")
		line = strings.TrimSpace(line)
		if line != "" {
			points = append(points, line)
		}
	}
	if len(points) > req.MaxPoints {
		points = points[:req.MaxPoints]
	}
	return points, nil
}

func (c *OpenAICompatClient) ExpandSection(ctx context.Context, req ExpandSectionRequest) (string, error) {
	const maxAttempts = 2
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		content, err := c.doExpandSection(ctx, req)
		if err == nil {
			return content, nil
		}
		lastErr = err
		if !isRetryable(err) || attempt == maxAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return "", fmt.Errorf("正文生成重试 %d 次后仍失败: %w", maxAttempts, lastErr)
}

func (c *OpenAICompatClient) doExpandSection(ctx context.Context, req ExpandSectionRequest) (string, error) {
	// 中文约 1.5–2 token/字，按 2 倍预算留足余量，避免长章节被截断
	maxTokens := req.WordBudget * 2
	if maxTokens < 2048 {
		maxTokens = 2048
	}
	if maxTokens > 32768 {
		maxTokens = 32768
	}
	return c.chat(ctx, buildExpandSectionPrompt(req), maxTokens)
}

func (c *OpenAICompatClient) chat(ctx context.Context, prompt string, maxTokens int) (string, error) {
	// 不设置 thinking 参数：新版智谱 glm-5.x 强制思考，传任何 thinking 值都返回 400
	// （"该模型始终思考，不支持关闭思考"）。让它用默认行为。
	body := map[string]interface{}{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  maxTokens,
	}
	raw, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("请求 LLM 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", &HTTPError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析 LLM 响应失败: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM 返回空 choices")
	}
	// C1：累加 token 用量
	c.addUsage(result.Usage.PromptTokens, result.Usage.CompletionTokens)
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}