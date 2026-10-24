package browse

import (
	"context"
	"fmt"
	"sync"
)

// MockBrowser 是 Browser 的内存实现，用于测试。
//
// 行为：
//   - Navigate 记录 URL，不碰网络
//   - ReadPage 返回预设内容
//   - 可在测试中注入错误，模拟各种失败场景
type MockBrowser struct {
	mu sync.Mutex

	// Pages 是 URL → 页面正文的映射。Navigate 时记录，ReadPage 时返回。
	Pages map[string]string

	// NavigateErr / ReadPageErr 非 nil 时，对应方法直接返回此错误。
	// 用于测试错误路径。
	NavigateErr error
	ReadPageErr error

	// Closed 记录 Close 是否被调用，供测试断言。
	Closed bool

	// LastURL 记录最后一次 Navigate 的 URL，供测试断言。
	LastURL string
}

// NewMockBrowser 创建 MockBrowser，Pages 初始化为空 map。
func NewMockBrowser() *MockBrowser {
	return &MockBrowser{Pages: make(map[string]string)}
}

func (m *MockBrowser) Navigate(ctx context.Context, rawURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.NavigateErr != nil {
		return m.NavigateErr
	}
	if rawURL == "" {
		return fmt.Errorf("browse: MockBrowser.Navigate 收到空 URL")
	}
	m.LastURL = rawURL
	if _, ok := m.Pages[rawURL]; !ok {
		m.Pages[rawURL] = "（Mock 页面：" + rawURL + "）"
	}
	return nil
}

func (m *MockBrowser) ReadPage(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ReadPageErr != nil {
		return "", m.ReadPageErr
	}
	if m.LastURL == "" {
		return "", fmt.Errorf("browse: MockBrowser.ReadPage 调用前未 Navigate")
	}
	return m.Pages[m.LastURL], nil
}

func (m *MockBrowser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Closed = true
	return nil
}
