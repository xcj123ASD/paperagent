package llm

import (
	"context"
	"errors"
	"testing"
)

// --- 测试辅助 ---

type okClient struct{ points []string }

func (o okClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	return o.points, nil
}

type failClient struct{ err error }

func (f failClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	return nil, f.err
}

// --- FallbackClient ---

func TestFallbackClient_PrimaryOK_NoFallback(t *testing.T) {
	backupCalled := false
	f := FallbackClient{
		Primary: okClient{points: []string{"p"}},
		Backup:  trackingClient{called: &backupCalled},
	}
	points, err := f.KeyPoints(context.Background(), KeyPointRequest{})
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(points) != 1 || points[0] != "p" {
		t.Errorf("应返回 Primary 结果，实际 %v", points)
	}
	if backupCalled {
		t.Error("Primary 成功时不应调用 Backup")
	}
}

func TestFallbackClient_PrimaryFails_UseBackup(t *testing.T) {
	f := FallbackClient{
		Primary: failClient{err: errors.New("boom")},
		Backup:  okClient{points: []string{"b"}},
	}
	points, err := f.KeyPoints(context.Background(), KeyPointRequest{})
	if err != nil {
		t.Fatalf("应降级成功: %v", err)
	}
	if len(points) != 1 || points[0] != "b" {
		t.Errorf("应返回 Backup 结果，实际 %v", points)
	}
}

func TestFallbackClient_BothFail_ReturnPrimaryErr(t *testing.T) {
	f := FallbackClient{
		Primary: failClient{err: errors.New("primary-err")},
		Backup:  failClient{err: errors.New("backup-err")},
	}
	_, err := f.KeyPoints(context.Background(), KeyPointRequest{})
	if err == nil {
		t.Fatal("两个都失败应报错")
	}
	if err.Error() != "backup-err" {
		t.Errorf("应返回 Backup 的错误，实际 %q", err.Error())
	}
}

func TestFallbackClient_NoBackup_ReturnPrimaryErr(t *testing.T) {
	f := FallbackClient{Primary: failClient{err: errors.New("x")}}
	_, err := f.KeyPoints(context.Background(), KeyPointRequest{})
	if err == nil {
		t.Fatal("Backup 为 nil 时应返回 Primary 错误")
	}
}

type trackingClient struct{ called *bool }

func (t trackingClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	*t.called = true
	return []string{"backup"}, nil
}

// --- NewFromEnv ---

func TestNewFromEnv_NoKeys_ReturnsNil(t *testing.T) {
	t.Setenv("DASHSCOPE_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	if c := NewFromEnv(); c != nil {
		t.Fatalf("无 Key 应返回 nil，实际 %T", c)
	}
}

func TestNewFromEnv_OnlyQwen(t *testing.T) {
	t.Setenv("DASHSCOPE_API_KEY", "qwen-key")
	t.Setenv("DEEPSEEK_API_KEY", "")
	c := NewFromEnv()
	oc, ok := c.(*OpenAICompatClient)
	if !ok {
		t.Fatalf("只设 qwen 应返回 *OpenAICompatClient，实际 %T", c)
	}
	if oc.Model != "qwen-flash" {
		t.Errorf("model 应为 qwen-flash，实际 %s", oc.Model)
	}
}

func TestNewFromEnv_OnlyDeepSeek(t *testing.T) {
	t.Setenv("DASHSCOPE_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "ds-key")
	c := NewFromEnv()
	oc, ok := c.(*OpenAICompatClient)
	if !ok {
		t.Fatalf("只设 deepseek 应返回 *OpenAICompatClient，实际 %T", c)
	}
	if oc.Model != "deepseek-flash" {
		t.Errorf("model 应为 deepseek-flash，实际 %s", oc.Model)
	}
}

func TestNewFromEnv_Both_QwenPrimary(t *testing.T) {
	t.Setenv("DASHSCOPE_API_KEY", "qwen-key")
	t.Setenv("DEEPSEEK_API_KEY", "ds-key")
	t.Setenv("LLM_PROVIDER", "qwen")
	c := NewFromEnv()
	fc, ok := c.(FallbackClient)
	if !ok {
		t.Fatalf("两个 Key 应返回 FallbackClient，实际 %T", c)
	}
	p := fc.Primary.(*OpenAICompatClient)
	b := fc.Backup.(*OpenAICompatClient)
	if p.Model != "qwen-flash" {
		t.Errorf("Primary 应为 qwen-flash，实际 %s", p.Model)
	}
	if b.Model != "deepseek-flash" {
		t.Errorf("Backup 应为 deepseek-flash，实际 %s", b.Model)
	}
}

func TestNewFromEnv_Both_DeepSeekPrimary(t *testing.T) {
	t.Setenv("DASHSCOPE_API_KEY", "qwen-key")
	t.Setenv("DEEPSEEK_API_KEY", "ds-key")
	t.Setenv("LLM_PROVIDER", "deepseek")
	c := NewFromEnv()
	fc, ok := c.(FallbackClient)
	if !ok {
		t.Fatalf("两个 Key 应返回 FallbackClient，实际 %T", c)
	}
	p := fc.Primary.(*OpenAICompatClient)
	b := fc.Backup.(*OpenAICompatClient)
	if p.Model != "deepseek-flash" {
		t.Errorf("Primary 应为 deepseek-flash，实际 %s", p.Model)
	}
	if b.Model != "qwen-flash" {
		t.Errorf("Backup 应为 qwen-flash，实际 %s", b.Model)
	}
}