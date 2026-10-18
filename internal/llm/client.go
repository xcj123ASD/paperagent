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

type MockClient struct{}

func (MockClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	return []string{
		req.Section + "：要点一（Mock）",
		req.Section + "：要点二（Mock）",
	}, nil
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

// NewFromEnv 根据环境变量选择 provider：
//   LLM_PROVIDER=qwen     -> 读 DASHSCOPE_API_KEY，模型 qwen-flash
//   LLM_PROVIDER=deepseek -> 读 DEEPSEEK_API_KEY，模型 deepseek-v4-flash
//   未设置或 key 为空      -> 返回 nil
func NewFromEnv() Client {
	provider := os.Getenv("LLM_PROVIDER")
	switch provider {
	case "qwen":
		key := os.Getenv("DASHSCOPE_API_KEY")
		if key == "" {
			return nil
		}
		return NewOpenAICompat(
			"https://dashscope.aliyuncs.com/compatible-mode/v1",
			key,
			"qwen-flash",
		)
	case "deepseek":
		key := os.Getenv("DEEPSEEK_API_KEY")
		if key == "" {
			return nil
		}
		return NewOpenAICompat(
			"https://api.deepseek.com",
			key,
			"deepseek-v4-flash",
		)
	default:
		return nil
	}
}

func (c *OpenAICompatClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	prompt := fmt.Sprintf(
		"你是一位学术论文写作助手。请为以下章节生成不超过 %d 个核心要点，每个要点一行，不要编号，不要解释。\n"+
			"论文主题：%s\n章节：%s\n章节作用：%s\n文章类型：%s",
		req.MaxPoints, req.Topic, req.Section, req.Role, req.ArticleType,
	)

	body := map[string]interface{}{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
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
		return nil, fmt.Errorf("LLM 返回 %d: %s", resp.StatusCode, string(respBody))
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