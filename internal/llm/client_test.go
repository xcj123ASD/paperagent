package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// --- 测试辅助 ---

type okClient struct{ points []string }

func (o okClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	return o.points, nil
}
func (o okClient) ExpandSection(ctx context.Context, req ExpandSectionRequest) (string, error) {
	return "ok-content", nil
}

type failClient struct{ err error }

func (f failClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	return nil, f.err
}
func (f failClient) ExpandSection(ctx context.Context, req ExpandSectionRequest) (string, error) {
	return "", f.err
}

type trackingClient struct{ called *bool }

func (t trackingClient) KeyPoints(ctx context.Context, req KeyPointRequest) ([]string, error) {
	*t.called = true
	return []string{"backup"}, nil
}
func (t trackingClient) ExpandSection(ctx context.Context, req ExpandSectionRequest) (string, error) {
	*t.called = true
	return "backup-content", nil
}

// --- FallbackClient: KeyPoints ---

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

func TestFallbackClient_BothFail_ReturnBackupErr(t *testing.T) {
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

// --- FallbackClient: ExpandSection ---

func TestFallbackClient_Expand_PrimaryOK(t *testing.T) {
	backupCalled := false
	f := FallbackClient{
		Primary: okClient{},
		Backup:  trackingClient{called: &backupCalled},
	}
	content, err := f.ExpandSection(context.Background(), ExpandSectionRequest{})
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if content != "ok-content" {
		t.Errorf("应返回 Primary 正文，实际 %q", content)
	}
	if backupCalled {
		t.Error("Primary 成功时不应调用 Backup")
	}
}

func TestFallbackClient_Expand_PrimaryFails_UseBackup(t *testing.T) {
	f := FallbackClient{
		Primary: failClient{err: errors.New("boom")},
		Backup:  okClient{},
	}
	content, err := f.ExpandSection(context.Background(), ExpandSectionRequest{})
	if err != nil {
		t.Fatalf("应降级成功: %v", err)
	}
	if content != "ok-content" {
		t.Errorf("应返回 Backup 正文，实际 %q", content)
	}
}

// --- MockClient ---

func TestMockClient_ExpandSection_RoughlyMatchesBudget(t *testing.T) {
	m := MockClient{}
	content, err := m.ExpandSection(context.Background(), ExpandSectionRequest{
		Section:    "引言",
		KeyPoints:  []string{"要点A", "要点B"},
		WordBudget: 200,
	})
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if content == "" {
		t.Fatal("正文不应为空")
	}
	if !strings.Contains(content, "引言") {
		t.Error("正文应包含章节名")
	}
	// 粗略检查：字数应与 WordBudget 同量级（0.3x ~ 3x）
	runes := len([]rune(content))
	if runes < 60 || runes > 600 {
		t.Errorf("字数偏离过大，WordBudget=200，实际=%d", runes)
	}
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
	if oc.Model != "deepseek-chat" {
		t.Errorf("model 应为 deepseek-chat，实际 %s", oc.Model)
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
	if b.Model != "deepseek-chat" {
		t.Errorf("Backup 应为 deepseek-chat，实际 %s", b.Model)
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
	if p.Model != "deepseek-chat" {
		t.Errorf("Primary 应为 deepseek-chat，实际 %s", p.Model)
	}
	if b.Model != "qwen-flash" {
		t.Errorf("Backup 应为 qwen-flash，实际 %s", b.Model)
	}
}