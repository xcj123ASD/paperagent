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
	"time"
)

type KeyPointRequest struct {
	Topic       string
	Section     string
	Role        string
	ArticleType string
	MaxPoints   int
}

type Client interface {
	KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error)
}

// MockClient 离线用，不依赖网络
type MockClient struct{}

func (MockClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	return []string{
		req.Section + "：要点一（Mock）",
		req.Section + "：要点二（Mock）",
	}, nil
}

// FallbackClient 主备降级：Primary 失败自动切 Backup。
// 任何错误都触发降级（包括 401），因为另一个 provider 的 Key 可能是对的。
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

type OpenAICompatClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func NewOpenAICompat(baseURL, apiKey, model string) *OpenAICompatClient {
	return &OpenAICompatClient{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewFromEnv 同时构建通义千问和 DeepSeek：
//   - 两个 Key 都设：返回 FallbackClient，主用 LLM_PROVIDER 指定的，挂了自动切另一个
//   - 只设一个 Key：返回那一个客户端，行为跟单 provider 一样
//   - 都没设：返回 nil，orchestrator 会 fallback 到 Mock
func NewFromEnv() Client {
	qwenKey := os.Getenv("DASHSCOPE_API_KEY")
	dsKey := os.Getenv("DEEPSEEK_API_KEY")

	var qwen, deepseek Client
	if qwenKey != "" {
		qwen = NewOpenAICompat(
			"https://dashscope.aliyuncs.com/compatible-mode/v1",
			qwenKey,
			"qwen-flash",
		)
	}
	if dsKey != "" {
		deepseek = NewOpenAICompat(
			"https://api.deepseek.com",
			dsKey,
			"deepseek-flash", // 官方推荐名，指向 V4.1-Flash-1017
		)
	}

	switch {
	case qwen != nil && deepseek != nil:
		if os.Getenv("LLM_PROVIDER") == "deepseek" {
			return FallbackClient{Primary: deepseek, Backup: qwen}
		}
		return FallbackClient{Primary: qwen, Backup: deepseek}
	case qwen != nil:
		return qwen
	case deepseek != nil:
		return deepseek
	default:
		return nil
	}
}

// HTTPError 带状态码，供重试逻辑区分可重试/不可重试
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("LLM 返回 %d: %s", e.StatusCode, e.Body)
}

// buildKeyPointPrompt 是 Skill 接入的锚点。
// M1 之后迁移到 SKILL.md 时，只改这一个函数。
func buildKeyPointPrompt(req KeyPointRequest) string {
	return fmt.Sprintf(
		"你是一位学术论文写作助手。请为以下章节生成不超过 %d 个核心要点，每个要点一行，不要编号，不要解释。\n"+
			"论文主题：%s\n章节：%s\n章节作用：%s\n文章类型：%s",
		req.MaxPoints, req.Topic, req.Section, req.Role, req.ArticleType,
	)
}

// KeyPoints 带重试：网络/5xx/429 重试 3 次；4xx（除 429）直接失败
func (c *OpenAICompatClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		points, err := c.doKeyPoints(ctx, req)
		if err == nil {
			return points, nil
		}
		lastErr = err
		if he, ok := err.(*HTTPError); ok {
			if he.StatusCode >= 400 && he.StatusCode < 500 && he.StatusCode != 429 {
				return nil, err
			}
		}
		if attempt == maxAttempts-1 {
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
	body := map[string]interface{}{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "user", "content": buildKeyPointPrompt(req)},
		},
		"temperature": 0.7,
		"max_tokens":  512,
	}
	raw, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 LLM 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 LLM 响应失败: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("LLM 返回空 choices")
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
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