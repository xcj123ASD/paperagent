package browse

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// 接口契约：*MockBrowser 必须满足 Browser 接口。
var _ Browser = (*MockBrowser)(nil)

func TestMockBrowser_NavigateAndRead(t *testing.T) {
	m := NewMockBrowser()
	ctx := context.Background()

	if err := m.Navigate(ctx, "https://example.com/"); err != nil {
		t.Fatalf("Navigate 失败: %v", err)
	}
	if m.LastURL != "https://example.com/" {
		t.Errorf("LastURL 应为 https://example.com/，实际 %q", m.LastURL)
	}

	text, err := m.ReadPage(ctx)
	if err != nil {
		t.Fatalf("ReadPage 失败: %v", err)
	}
	if !strings.Contains(text, "example.com") {
		t.Errorf("ReadPage 应含 URL，实际: %q", text)
	}
}

func TestMockBrowser_NavigateEmptyURL(t *testing.T) {
	m := NewMockBrowser()
	if err := m.Navigate(context.Background(), ""); err == nil {
		t.Fatal("空 URL 应报错")
	}
}

func TestMockBrowser_ReadBeforeNavigate(t *testing.T) {
	m := NewMockBrowser()
	if _, err := m.ReadPage(context.Background()); err == nil {
		t.Fatal("Navigate 前 ReadPage 应报错")
	}
}

func TestMockBrowser_NavigateErrInjection(t *testing.T) {
	m := NewMockBrowser()
	wantErr := errors.New("模拟导航失败")
	m.NavigateErr = wantErr
	err := m.Navigate(context.Background(), "https://example.com/")
	if !errors.Is(err, wantErr) {
		t.Errorf("应返回注入的错误，实际: %v", err)
	}
}

func TestMockBrowser_ReadPageErrInjection(t *testing.T) {
	m := NewMockBrowser()
	_ = m.Navigate(context.Background(), "https://example.com/")
	wantErr := errors.New("模拟读取失败")
	m.ReadPageErr = wantErr
	_, err := m.ReadPage(context.Background())
	if !errors.Is(err, wantErr) {
		t.Errorf("应返回注入的错误，实际: %v", err)
	}
}

func TestMockBrowser_Close(t *testing.T) {
	m := NewMockBrowser()
	if m.Closed {
		t.Fatal("初始应为未关闭")
	}
	if err := m.Close(); err != nil {
		t.Fatalf("Close 失败: %v", err)
	}
	if !m.Closed {
		t.Error("Close 后 Closed 应为 true")
	}
	// 幂等：重复 Close 不报错
	if err := m.Close(); err != nil {
		t.Errorf("重复 Close 应幂等，实际: %v", err)
	}
}

func TestMockBrowser_CustomPageContent(t *testing.T) {
	m := NewMockBrowser()
	m.Pages["https://test.local/"] = "自定义内容"
	ctx := context.Background()
	_ = m.Navigate(ctx, "https://test.local/")
	text, err := m.ReadPage(ctx)
	if err != nil {
		t.Fatalf("ReadPage 失败: %v", err)
	}
	if text != "自定义内容" {
		t.Errorf("应返回预设内容，实际: %q", text)
	}
}
